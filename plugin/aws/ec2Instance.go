package aws

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/kballard/go-shellquote"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/volume"
)

// ec2Instance represents an EC2 instance
type ec2Instance struct {
	plugin.EntryBase
	id                      string
	session                 *session.Session
	client                  *ec2Client.EC2
	ssmClient               *ssm.SSM
	cloudwatchClient        *cloudwatchlogs.CloudWatchLogs
	latestConsoleOutputOnce sync.Once
	hasLatestConsoleOutput  bool
}

// These constants represent the possible states that the EC2 instance
// could be in. We export these constants so that other packages could
// use them since they are not provided by the AWS SDK.
const (
	EC2InstancePendingState      = 0
	EC2InstanceRunningState      = 16
	EC2InstanceShuttingDownState = 32
	EC2InstanceTerminated        = 48
	EC2InstanceStopping          = 64
	EC2InstanceStopped           = 80
)

func newEC2Instance(ctx context.Context, inst *ec2Client.Instance, session *session.Session, client *ec2Client.EC2) *ec2Instance {
	id := awsSDK.StringValue(inst.InstanceId)
	name := id
	// AWS has a practice of using a tag with the key 'Name' as the display name in the console, so
	// it's common for resources to be given a (non-unique) name. Use that to mimic the console, but
	// append the instance ID to ensure it's unique. We start with name so that things with the same
	// name will be grouped when sorted.
	for _, tag := range inst.Tags {
		if awsSDK.StringValue(tag.Key) == "Name" {
			name = awsSDK.StringValue(tag.Value) + "_" + id
			break
		}
	}
	ec2Instance := &ec2Instance{
		EntryBase:        plugin.NewEntry(name),
		id:               id,
		session:          session,
		client:           client,
		ssmClient:        ssm.New(session),
		cloudwatchClient: cloudwatchlogs.New(session),
	}
	ec2Instance.SetTTLOf(plugin.ListOp, 30*time.Second)
	ec2Instance.DisableCachingFor(plugin.MetadataOp)

	metaObj := newDescribeInstanceResult(inst)

	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(metaObj.ctime).
		SetMtime(metaObj.mtime).
		SetMeta(metaObj.toMeta())
	ec2Instance.SetAttributes(attr)

	return ec2Instance
}

type describeInstanceResult struct {
	inst  *ec2Client.Instance
	ctime time.Time
	mtime time.Time
}

func newDescribeInstanceResult(inst *ec2Client.Instance) describeInstanceResult {
	result := describeInstanceResult{
		inst: inst,
	}

	// AWS does not include the EC2 instance's ctime in its
	// metadata. It also does not include the EC2 instance's
	// last state transition time (mtime). Thus, we try to "guess"
	// reasonable values for ctime and mtime by looping over each
	// block device's attachment time and the instance's launch time.
	// The oldest of these times is the ctime; the newest is the mtime.
	result.ctime = awsSDK.TimeValue(inst.LaunchTime)
	result.mtime = result.ctime
	for _, mapping := range inst.BlockDeviceMappings {
		attachTime := awsSDK.TimeValue(mapping.Ebs.AttachTime)

		if attachTime.Before(result.ctime) {
			result.ctime = attachTime
		}

		if attachTime.After(result.mtime) {
			result.mtime = attachTime
		}
	}

	return result
}

func (d describeInstanceResult) toMeta() plugin.EntryMetadata {
	meta := plugin.ToMeta(d.inst)
	meta["CreationTime"] = d.ctime
	meta["LastModifiedTime"] = d.mtime

	return meta
}

func (inst *ec2Instance) cachedDescribeInstance(ctx context.Context) (describeInstanceResult, error) {
	info, err := plugin.CachedOp(ctx, "DescribeInstance", inst, 15*time.Second, func() (interface{}, error) {
		request := &ec2Client.DescribeInstancesInput{
			InstanceIds: []*string{
				awsSDK.String(inst.id),
			},
		}

		resp, err := inst.client.DescribeInstances(request)
		if err != nil {
			return nil, err
		}

		inst := resp.Reservations[0].Instances[0]
		return newDescribeInstanceResult(inst), nil
	})

	if err != nil {
		return describeInstanceResult{}, err
	}

	return info.(describeInstanceResult), nil
}

func (inst *ec2Instance) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	result, err := inst.cachedDescribeInstance(ctx)
	if err != nil {
		return nil, err
	}

	return result.toMeta(), nil
}

func (inst *ec2Instance) List(ctx context.Context) ([]plugin.Entry, error) {
	var latestConsoleOutput *ec2InstanceConsoleOutput
	var err error
	inst.latestConsoleOutputOnce.Do(func() {
		latestConsoleOutput, err = inst.checkLatestConsoleOutput(ctx)
	})

	entries := []plugin.Entry{}

	metadataJSON, err := newEC2InstanceMetadataJSON(ctx, inst)
	if err != nil {
		return nil, err
	}
	entries = append(entries, metadataJSON)

	consoleOutput, err := newEC2InstanceConsoleOutput(ctx, inst, false)
	if err != nil {
		return nil, err
	}
	entries = append(entries, consoleOutput)

	if inst.hasLatestConsoleOutput {
		if latestConsoleOutput == nil {
			latestConsoleOutput, err = newEC2InstanceConsoleOutput(ctx, inst, true)
			if err != nil {
				return nil, err
			}
		}
		entries = append(entries, latestConsoleOutput)
	}

	// Include a view of the remote filesystem using volume.FS
	entries = append(entries, volume.NewFS("fs", inst))

	return entries, nil
}

// According to https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-console.html,
// only instance types that use the Nitro hypervisor can retrieve the
// latest console output. For all other instance types, AWS will return
// an unsupported operation error when they attempt to get the latest
// console output. Thus, this checks to see if our EC2 instance supports retrieving
// the console logs, which reduces to checking whether we can open a
// consoleLatestOutput object.
//
// NOTE: We return the object to avoid an extra request in List. The returned error
// is whether something went wrong with opening the consoleLatestOutput object (so
// that List can appropriately error).
func (inst *ec2Instance) checkLatestConsoleOutput(ctx context.Context) (*ec2InstanceConsoleOutput, error) {
	consoleLatestOutput, err := newEC2InstanceConsoleOutput(ctx, inst, true)
	if err == nil {
		inst.hasLatestConsoleOutput = true
		return consoleLatestOutput, nil
	}

	awserr, ok := err.(awserr.Error)
	if !ok {
		// Open failed w/ some other error, which should be a
		// rare occurrence. Here we reset latestConsoleOutputOnce
		// so that we check again for the latest console output the
		// next time List's called, then return an error
		inst.latestConsoleOutputOnce = sync.Once{}
		return nil, fmt.Errorf(
			"could not determine whether the EC2 instance %v supports retrieving the latest console output: %v",
			inst.Name(),
			ctx.Err(),
		)
	}

	// For some reason, the EC2 client does not have this error code
	// as a constant.
	if awserr.Code() == "UnsupportedOperation" {
		inst.hasLatestConsoleOutput = false
		return nil, nil
	}

	// Open failed due to some other AWS-related error. Assume this means
	// that the instance _does_ have the latest console logs, but something
	// went wrong with accessing them.
	inst.hasLatestConsoleOutput = true
	return nil, fmt.Errorf("could not access the latest console log: %v", err)
}

// These constants represent the possible states that the command
// could be in. They're needed by Exec. We export these constants
// so that other packages could use them since they are not provided
// by the AWS SDK.
//
// See https://docs.aws.amazon.com/systems-manager/latest/userguide/monitor-commands.html
// for a detailed explanation of each of these states.
const (
	CommandPending           = "Pending"
	CommandInProgress        = "InProgress"
	CommandDelayed           = "Delayed"
	CommandSuccess           = "Success"
	CommandDeliveryTimedOut  = "DeliveryTimedOut"
	CommandExecutionTimedOut = "ExecutionTimedOut"
	CommandFailed            = "Failed"
	CommandCanceled          = "Canceled"
	CommandUndeliverable     = "Undeliverable"
	CommandTerminated        = "Terminated"
)

func (inst *ec2Instance) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecResult, error) {
	execResult := plugin.ExecResult{}

	result, err := inst.cachedDescribeInstance(ctx)
	if err != nil {
		return execResult, nil
	}

	// Exec only makes sense on a running EC2 instance.
	if awsSDK.Int64Value(result.inst.State.Code) != EC2InstanceRunningState {
		err := fmt.Errorf(
			"instance is not in the running state. Its current state is: %v",
			awsSDK.StringValue(result.inst.State.Name),
		)

		return execResult, err
	}

	// We're running our command as a shell script so we'll need to do
	// some shell escaping.
	cmdStr := shellquote.Join(
		append([]string{cmd}, args...)...,
	)
	if opts.Stdin != nil {
		input, err := ioutil.ReadAll(opts.Stdin)
		if err != nil {
			return execResult, fmt.Errorf("failed to read stdin: %v", err)
		}

		// Handle Stdin by piping its content into our command.
		// Note that this won't work well for large Stdin streams.
		quoted := shellquote.Join(string(input))
		// Add extra escaping needed for some shells. This should probably be part of shellquote.
		quoted = strings.Replace(quoted, "\\t", "\\\\t", -1)
		quoted = strings.Replace(quoted, "\\n", "\\\\n", -1)
		quoted = strings.Replace(quoted, "\\0", "\\\\0", -1)
		cmdStr = "echo -n " + quoted + " | " + cmdStr
	}

	activity.Record(ctx, "Sending the following command to the SSM agent:\n%v", cmdStr)
	request := &ssm.SendCommandInput{
		CloudWatchOutputConfig: &ssm.CloudWatchOutputConfig{
			// When CloudWatch output is enabled, AWS will send stdout/stderr
			// to the CloudWatch logs.
			CloudWatchOutputEnabled: awsSDK.Bool(true),
		},
		Comment: awsSDK.String("Document triggered by Wash"),
		// TODO: AWS-RunShellScript only works on Linux instances. We'll need to use
		// AWS-RunPowerShellScript for Windows.
		DocumentName: awsSDK.String("AWS-RunShellScript"),
		InstanceIds: []*string{
			awsSDK.String(inst.id),
		},
		// See https://docs.aws.amazon.com/systems-manager/latest/userguide/ssm-plugins.html#aws-runShellScript
		// for a list of all the parameters that AWS-RunShellScript
		// supports. Note that there are some naming inconsistencies
		// between the parameter names, so here is a mapping of
		// the parameter name in the doc. vs. what you'd pass-in
		// to the request:
		//     "runCommand"       => "commands"
		//     "timeoutSeconds"   => "executionTimeout"
		//     "workingDirectory" => "workingDirectory"
		Parameters: map[string][]*string{
			"commands": awsSDK.StringSlice(
				[]string{cmdStr},
			),
		},
		// The minimum value is 30 seconds. This guarantees that if our
		// command has not yet started executing, then it will not run.
		TimeoutSeconds: awsSDK.Int64(30),
	}
	resp, err := inst.ssmClient.SendCommandWithContext(ctx, request)
	if err != nil {
		return execResult, err
	}

	commandID := awsSDK.StringValue(resp.Command.CommandId)
	activity.Record(ctx, "Successfully sent the command. Command ID: %v", commandID)

	outputCh, outputStreamer := newOutputStreamer(ctx, inst.cloudwatchClient, commandID, inst.id)
	var exitCode int
	var exitCodeErr error
	go func() {
		// Some helpful functions
		closeOutputStreamer := func(err error) {
			outputStreamer.CloseWithError(err)
			exitCodeErr = err
		}
		cancelCommand := func(reason string) {
			activity.Record(ctx, "Cancelling the command. Reason: %v", reason)

			request := &ssm.CancelCommandInput{
				CommandId: awsSDK.String(commandID),
				InstanceIds: awsSDK.StringSlice(
					[]string{inst.id},
				),
			}

			// Use a different context to cancel the command so that Wash
			// will still attempt to cancel it when our current context is
			// cancelled (i.e. when reason == ctx.Err().Error()).
			cancelCtx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelFunc()

			// CancelCommand's response doesn't contain any useful information,
			// so we can discard it.
			_, err := inst.ssmClient.CancelCommandWithContext(cancelCtx, request)
			if err != nil {
				activity.Record(ctx, "Failed to cancel the command: %v", err)
			} else {
				activity.Record(ctx, "Successfully cancelled the command")
			}
		}

		// According to https://github.com/aws/amazon-ssm-agent/blob/2.3.479.0/agent/agentlogstocloudwatch/cloudwatchlogspublisher/cloudwatchlogsservice.go#L40,
		// SSM updates the stdout/stderr logs every three seconds,
		// so we'll sleep a little bit more than that between each
		// iteration to account for delays
		sleepDuration := 4 * time.Second

		for {
			time.Sleep(sleepDuration)

			activity.Record(ctx, "Getting the command status")
			request := &ssm.GetCommandInvocationInput{
				CommandId:  awsSDK.String(commandID),
				InstanceId: awsSDK.String(inst.id),
			}

			resp, err := inst.ssmClient.GetCommandInvocationWithContext(ctx, request)
			if err != nil {
				err := fmt.Errorf("failed to get the command status: %v", err)
				cancelCommand(err.Error())
				closeOutputStreamer(err)
				return
			}

			status := awsSDK.StringValue(resp.StatusDetails)
			activity.Record(ctx, "Command status: %v", status)

			switch status {
			case CommandPending, CommandDelayed:
				continue
			case CommandInProgress:
				if _, err := outputStreamer.sendNextChunks(); err != nil {
					cancelCommand(err.Error())
					closeOutputStreamer(err)
					return
				}
			case CommandSuccess, CommandFailed:
				exitCode = int(awsSDK.Int64Value(resp.ResponseCode))

				// Write the remaining output chunks (if any)
				activity.Record(ctx, "Command finished. Sending over its remaining output...")
				for {
					chunksWereSent, err := outputStreamer.sendNextChunks()
					if err != nil {
						activity.Record(ctx, "Failed to send over the remaining output: %v", err)
						closeOutputStreamer(err)
						return
					}

					if chunksWereSent {
						time.Sleep(sleepDuration)
					} else {
						// No new chunks were written. Assume that this means
						// we've finished writing the output
						activity.Record(ctx, "Finished sending the output.")
						closeOutputStreamer(nil)
						return
					}
				}
			case CommandExecutionTimedOut:
				closeOutputStreamer(fmt.Errorf("the command timed out during its execution"))
				return
			default:
				closeOutputStreamer(fmt.Errorf("the command failed to start. Its status is %v", status))
				return
			}
		}
	}()

	execResult.OutputCh = outputCh
	execResult.ExitCodeCB = func() (int, error) {
		if exitCodeErr != nil {
			return 0, exitCodeErr
		}

		return exitCode, nil
	}

	return execResult, nil
}

// This streamer fetches new CloudWatch log events from the stdout/stderr logs,
// and writes them to the respective stdout/stderr stream in timestamp order.
type outputStreamer struct {
	ctx             context.Context
	stdout          *plugin.OutputStream
	stdoutLogStream *outputLogStream
	stderr          *plugin.OutputStream
	stderrLogStream *outputLogStream
}

func newOutputStreamer(
	ctx context.Context,
	client *cloudwatchlogs.CloudWatchLogs,
	commandID string,
	instanceID string,
) (<-chan plugin.ExecOutputChunk, *outputStreamer) {
	s := &outputStreamer{
		ctx:             ctx,
		stdoutLogStream: newOutputLogStream("stdout", client, commandID, instanceID),
		stderrLogStream: newOutputLogStream("stderr", client, commandID, instanceID),
	}

	outputCh, stdout, stderr := plugin.CreateExecOutputStreams(ctx)
	s.stdout = stdout
	s.stderr = stderr

	return outputCh, s
}

func (s *outputStreamer) sendNextChunks() (bool, error) {
	activity.Record(s.ctx, "Attempting to send the next set of output chunks to stdout and stderr")

	// Unfortunately, we can't use FilterLogEvents to get this info in a single
	// stream because AWS gives no guarantee that the interleaved events will be
	// properly sorted.
	//
	// NOTE: According to https://docs.aws.amazon.com/cli/latest/reference/logs/get-log-events.html,
	// the largest each response can be is 1 MB or 10000 events, so worst-case we will
	// get 2 MB of information when fetching stdout/stderr events. Probably not a big
	// deal right now, but maybe an issue in the future if/when we do a parallel exec.
	stdoutEvents, err := s.stdoutLogStream.nextEvents(s.ctx)
	if err != nil {
		return false, err
	}

	stderrEvents, err := s.stderrLogStream.nextEvents(s.ctx)
	if err != nil {
		return false, err
	}

	if len(stdoutEvents) == 0 && len(stderrEvents) == 0 {
		activity.Record(s.ctx, "Did not send any chunks: the stdout/stderr log streams did not have any new events")
		return false, nil
	}

	// We have some output chunks that we can send. Start by iterating over
	// stdout/stderr events, sending the earliest event first. Keep doing
	// this until we run out of stdout OR stderr events.
	stdoutIx := 0
	stderrIx := 0
	for {
		if stdoutIx >= len(stdoutEvents) || stderrIx >= len(stderrEvents) {
			break
		}

		stdoutEvent := stdoutEvents[stdoutIx]
		stderrEvent := stderrEvents[stderrIx]
		if timestampOf(stdoutEvent).Before(timestampOf(stderrEvent)) {
			if err := sendEvent(stdoutEvent, s.stdout); err != nil {
				return false, err
			}

			stdoutIx++
		} else {
			if err := sendEvent(stderrEvent, s.stderr); err != nil {
				return false, err
			}

			stderrIx++
		}
	}

	// Here, at least one of stdoutEvents/stderrEvents is
	// empty.
	if stdoutIx >= len(stdoutEvents) {
		// Send any remaining stderr events,
		for ; stderrIx < len(stderrEvents); stderrIx++ {
			if err := sendEvent(stderrEvents[stderrIx], s.stderr); err != nil {
				return false, err
			}
		}
	} else {
		// Send any remaining stdout events
		for ; stdoutIx < len(stdoutEvents); stdoutIx++ {
			if err := sendEvent(stdoutEvents[stdoutIx], s.stdout); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

func (s *outputStreamer) CloseWithError(err error) {
	s.stdout.CloseWithError(err)
	s.stderr.CloseWithError(err)
	s.stdoutLogStream.Close(s.ctx)
	s.stderrLogStream.Close(s.ctx)
}

func timestampOf(event *cloudwatchlogs.OutputLogEvent) time.Time {
	return time.Unix(0, int64(time.Millisecond)*awsSDK.Int64Value(event.Timestamp))
}

func sendEvent(e *cloudwatchlogs.OutputLogEvent, stream *plugin.OutputStream) error {
	timestamp := timestampOf(e)
	// SSM sends the messages line-by line; however, it does not append
	// the newline character to the last message in the event.
	//
	// NOTE: Since SSM uses bufio.Scanner to parse each message, there's a
	// chance that the printed stdout/stderr stream can end in an extra
	// newline character. Unfortunately, we don't have an easy way of determining
	// whether the last message did end in a new line, so let this be a known
	// issue until there's a good reason for us to address it.
	message := awsSDK.StringValue(e.Message) + "\n"

	return stream.WriteWithTimestamp(timestamp, []byte(message))
}

type outputLogStream struct {
	name          string
	created       bool
	nextToken     string
	client        *cloudwatchlogs.CloudWatchLogs
	logGroupName  string
	logStreamName string
}

func newOutputLogStream(
	name string,
	client *cloudwatchlogs.CloudWatchLogs,
	commandID string,
	instanceID string,
) *outputLogStream {
	return &outputLogStream{
		name:         name,
		created:      false,
		nextToken:    "",
		client:       client,
		logGroupName: "/aws/ssm/AWS-RunShellScript",
		logStreamName: fmt.Sprintf(
			"%v/%v/aws-runShellScript/%v",
			commandID,
			instanceID,
			name,
		),
	}
}

func (s *outputLogStream) nextEvents(ctx context.Context) ([]*cloudwatchlogs.OutputLogEvent, error) {
	activity.Record(
		ctx,
		"Fetching the next CloudWatch log events for %v. Log group name: %v. Log stream name: %v",
		s.name,
		s.logGroupName,
		s.logStreamName,
	)

	request := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  awsSDK.String(s.logGroupName),
		LogStreamName: awsSDK.String(s.logStreamName),
	}
	if s.nextToken == "" {
		// This is our first request
		request.StartFromHead = awsSDK.Bool(true)
	} else {
		request.NextToken = awsSDK.String(s.nextToken)
	}

	resp, err := s.client.GetLogEventsWithContext(ctx, request)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == cloudwatchlogs.ErrCodeResourceNotFoundException {
				if s.created {
					return nil, fmt.Errorf("the CloudWatch logs for %v were unexpectedly deleted", s.name)
				}

				// The stream hasn't been created. This is OK. It means that
				// AWS hasn't sent them over yet, OR the command hasn't printed
				// anything to this stream.
				activity.Record(
					ctx,
					"Did not fetch any events for %v: the log stream hasn't been created yet",
					s.name,
				)

				return []*cloudwatchlogs.OutputLogEvent{}, nil
			}
		}

		return nil, fmt.Errorf("failed to fetch the events for %v: %v", s.name, err)
	}
	s.created = true
	s.nextToken = awsSDK.StringValue(resp.NextForwardToken)

	activity.Record(ctx, "Fetched %v new events for %v", len(resp.Events), s.name)
	return resp.Events, nil
}

func (s *outputLogStream) Close(ctx context.Context) {
	if !s.created {
		return
	}

	activity.Record(
		ctx,
		"Deleting the logs for %v. Log group name: %v. Log stream name: %v",
		s.name,
		s.logGroupName,
		s.logStreamName,
	)

	request := &cloudwatchlogs.DeleteLogStreamInput{
		LogGroupName:  awsSDK.String(s.logGroupName),
		LogStreamName: awsSDK.String(s.logStreamName),
	}

	// Use a different context to delete the log stream so that Wash
	// will still attempt to delete it when our current context is
	// cancelled
	deleteCtx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFunc()

	// DeleteLogStream's response does not contain any useful information
	// so we can discard it
	_, err := s.client.DeleteLogStreamWithContext(deleteCtx, request)
	if err != nil {
		activity.Record(ctx, "Failed to delete the logs for %v: %v", s.name, err)
	}
}

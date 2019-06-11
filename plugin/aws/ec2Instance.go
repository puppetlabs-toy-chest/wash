package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/transport"
	"github.com/puppetlabs/wash/volume"
)

// ec2Instance represents an EC2 instance
type ec2Instance struct {
	plugin.EntryBase
	id                      string
	session                 *session.Session
	client                  *ec2Client.EC2
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

func ec2InstanceBase() *ec2Instance {
	ec2Instance := &ec2Instance{
		EntryBase: plugin.NewEntryBase(),
	}
	ec2Instance.
		SetLabel("instance").
		SetTTLOf(plugin.ListOp, 30*time.Second).
		DisableCachingFor(plugin.MetadataOp)
	return ec2Instance
}

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
	ec2Instance := ec2InstanceBase()
	ec2Instance.id = id
	ec2Instance.session = session
	ec2Instance.client = client
	ec2Instance.
		SetName(name).
		SetAttributes(getAttributes(inst))

	return ec2Instance
}

func getAttributes(inst *ec2Client.Instance) plugin.EntryAttributes {
	attr := plugin.EntryAttributes{}

	// AWS does not include the EC2 instance's ctime in its
	// metadata. It also does not include the EC2 instance's
	// last state transition time (mtime). Thus, we try to "guess"
	// reasonable values for ctime and mtime by looping over each
	// block device's attachment time and the instance's launch time.
	// The oldest of these times is the ctime; the newest is the mtime.
	ctime := awsSDK.TimeValue(inst.LaunchTime)
	mtime := ctime
	for _, mapping := range inst.BlockDeviceMappings {
		attachTime := awsSDK.TimeValue(mapping.Ebs.AttachTime)

		if attachTime.Before(ctime) {
			ctime = attachTime
		}

		if attachTime.After(mtime) {
			mtime = attachTime
		}
	}

	meta := plugin.ToJSONObject(inst)
	meta["CreationTime"] = ctime
	meta["LastModifiedTime"] = mtime

	attr.
		SetCtime(ctime).
		SetMtime(mtime).
		SetMeta(meta)

	return attr
}

func (inst *ec2Instance) ChildSchemas() []plugin.EntrySchema {
	return plugin.ChildSchemas(
		ec2InstanceConsoleOutputBase(),
		ec2InstanceMetadataJSONBase(),
		volume.FSBase(),
	)
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

func (inst *ec2Instance) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	meta, err := inst.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: scrape default user and authorized keys from console output. Probably only works for Amazon AMIs.
	// See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/connection-prereqs.html#connection-prereqs-fingerprint
	// for some helpful defaults we could add.
	var hostname string
	if name, ok := meta["PublicDnsName"]; ok {
		hostname = name.(string)
	} else if ipaddr, ok := meta["PublicIpAddress"]; ok {
		hostname = ipaddr.(string)
	} else {
		return nil, fmt.Errorf("No public interface found for %v", inst)
	}

	// Use the default user for Amazon AMIs. See above for ideas on making this more general. Can be
	// overridden in ~/.ssh/config.
	return transport.ExecSSH(ctx, transport.Identity{Host: hostname, User: "ec2-user"}, append([]string{cmd}, args...), opts)
}

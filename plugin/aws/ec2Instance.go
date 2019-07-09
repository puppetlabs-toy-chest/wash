package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/puppetlabs/wash/activity"
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
		EntryBase: plugin.NewEntry(name),
	}
	ec2Instance.id = id
	ec2Instance.session = session
	ec2Instance.client = client
	ec2Instance.
		SetTTLOf(plugin.ListOp, 30*time.Second).
		SetAttributes(getAttributes(inst))

	return ec2Instance
}

func (inst *ec2Instance) cachedConsoleOutput(ctx context.Context, latest bool) (consoleOutput, error) {
	var opname string
	if latest {
		opname = "ConsoleOutputLatest"
	} else {
		opname = "ConsoleOutput"
	}
	output, err := plugin.CachedOp(ctx, opname, inst, 30*time.Second, func() (interface{}, error) {
		request := &ec2Client.GetConsoleOutputInput{
			InstanceId: awsSDK.String(inst.id),
		}
		if latest {
			request.Latest = awsSDK.Bool(latest)
		}

		resp, err := inst.client.GetConsoleOutputWithContext(ctx, request)
		if err != nil {
			return nil, err
		}

		content, err := base64.StdEncoding.DecodeString(awsSDK.StringValue(resp.Output))
		if err != nil {
			return nil, err
		}

		return consoleOutput{
			mtime:   awsSDK.TimeValue(resp.Timestamp),
			content: content,
		}, nil
	})

	if err != nil {
		return consoleOutput{}, err
	}

	return output.(consoleOutput), nil
}

type ec2InstanceMetadata struct {
	*ec2Client.Instance
	CreationTime     time.Time
	LastModifiedTime time.Time
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

	meta := plugin.ToJSONObject(ec2InstanceMetadata{
		Instance:         inst,
		CreationTime:     ctime,
		LastModifiedTime: mtime,
	})

	attr.
		SetCtime(ctime).
		SetMtime(mtime).
		SetMeta(meta)

	return attr
}

func (inst *ec2Instance) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(inst, "instance").
		SetMetaAttributeSchema(ec2InstanceMetadata{})
}

func (inst *ec2Instance) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&ec2InstanceConsoleOutput{}).Schema(),
		(&ec2InstanceMetadataJSON{}).Schema(),
		(&volume.FS{}).Schema(),
	}
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

	// Include a view of the remote filesystem using volume.FS. Use a small maxdepth because
	// VMs can have lots of files and SSH is fast.
	entries = append(entries, volume.NewFS("fs", inst, 3))

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
	var hostname string
	if name, ok := meta["PublicDnsName"]; ok {
		hostname = name.(string)
	} else if ipaddr, ok := meta["PublicIpAddress"]; ok {
		hostname = ipaddr.(string)
	} else {
		return nil, fmt.Errorf("No public interface found for %v", inst)
	}

	var identityfile string
	if keyname, ok := meta["KeyName"]; ok {
		if homedir, err := os.UserHomeDir(); err != nil {
			activity.Record(ctx, "Cannot determine home directory for location of key file. But key name is "+keyname.(string)+" %v", err)
		} else {
			identityfile = (filepath.Join(homedir, ".ssh", (keyname.(string) + ".pem")))
		}
	}

	var fallbackuser string
	// Scan console output for user name instance was provisioned with. Set to ec2-user if not found
	re := regexp.MustCompile(`\WAuthorized keys from .home.*authorized_keys for user ([^+]*)+`)
	output, err := (inst.cachedConsoleOutput(ctx, inst.hasLatestConsoleOutput))
	if err != nil {
		activity.Record(ctx, "Cannot get cached console output: %v", err)
		fallbackuser = "ec2-user"
	} else {
		match := re.FindStringSubmatch(string(output.content))
		if match != nil {
			fallbackuser = (match[1])
		} else {
			activity.Record(ctx, "Cannot find provisioned user name in console output: %v", err)
			fallbackuser = "ec2-user"
		}
	}
	//
	// fallbackuser and identiyfile can be overridden in ~/.ssh/config.
	//
	return transport.ExecSSH(ctx, transport.Identity{Host: hostname, FallbackUser: fallbackuser, IdentityFile: identityfile}, append([]string{cmd}, args...), opts)
}

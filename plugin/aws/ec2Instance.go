package aws

import (
	"context"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

// ec2Instance represents an EC2 instance
type ec2Instance struct {
	plugin.EntryBase
	session *session.Session
	client  *ec2Client.EC2
	attr    plugin.Attributes
	entries []plugin.Entry
}

func newEC2Instance(ctx context.Context, ID string, session *session.Session, client *ec2Client.EC2, attr plugin.Attributes) *ec2Instance {
	ec2Instance := &ec2Instance{
		EntryBase: plugin.NewEntry(ID),
		session:   session,
		client:    client,
		attr:      attr,
	}
	ec2Instance.CacheConfig().TurnOffCachingFor(plugin.List)

	ec2Instance.entries = []plugin.Entry{
		newEC2InstanceMetadataJSON(ec2Instance),
		newEC2InstanceConsoleOutput(ec2Instance, false),
	}

	if ec2Instance.hasLatestConsoleOutput(ctx) {
		ec2Instance.entries = append(ec2Instance.entries, newEC2InstanceConsoleOutput(ec2Instance, true))
	}

	return ec2Instance
}

// According to https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-console.html,
// only instance types that use the Nitro hypervisor can retrieve the
// latest console output. For all other instance types, AWS will return
// an unsupported operation error when they attempt to get the latest
// console output. Thus, this checks to see if our EC2 instance supports retrieving
// the console logs, which reduces to checking whether we can open a
// consoleLatestOutput object.
func (inst *ec2Instance) hasLatestConsoleOutput(ctx context.Context) bool {
	consoleLatestOutput := newEC2InstanceConsoleOutput(inst, true)

	_, err := consoleLatestOutput.Open(ctx)
	if err == nil {
		return true
	}

	awserr, ok := err.(awserr.Error)
	if !ok {
		// Open failed w/ some other error, so log a warning and
		// return false. This should be a rare occurrence.
		log.Warnf(
			"could not determine whether the EC2 instance %v supports retrieving the latest console logs: %v",
			inst.Name(),
			ctx.Err(),
		)

		return false
	}

	// For some reason, the EC2 client does not have this error code
	// as a constant.
	if awserr.Code() == "UnsupportedOperation" {
		return false
	}

	// Open failed due to some other AWS-related error. Assume this means
	// that the instance _does_ have the latest console logs, but something
	// went wrong with accessing them.
	return true
}

func (inst *ec2Instance) Attr() plugin.Attributes {
	return inst.attr
}

func (inst *ec2Instance) Metadata(context.Context) (plugin.MetadataMap, error) {
	request := &ec2Client.DescribeInstancesInput{
		InstanceIds: []*string{
			awsSDK.String(inst.Name()),
		},
	}

	resp, err := inst.client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}

	// The API returns an error for an invalid instance ID. Since
	// our API call succeeded, the response is guaranteed to contain
	// the instance's metadata
	return plugin.ToMetadata(resp.Reservations[0].Instances[0]), nil
}

func (inst *ec2Instance) List(ctx context.Context) ([]plugin.Entry, error) {
	return inst.entries, nil
}

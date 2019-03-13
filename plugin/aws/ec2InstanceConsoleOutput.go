package aws

import (
	"bytes"
	"context"
	"encoding/base64"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	ec2Client "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/puppetlabs/wash/plugin"
)

// ec2InstanceConsoleOutput represents an EC2 instance's
// console output
type ec2InstanceConsoleOutput struct {
	plugin.EntryBase
	inst   *ec2Instance
	latest bool
}

func newEC2InstanceConsoleOutput(inst *ec2Instance, latest bool) *ec2InstanceConsoleOutput {
	cl := &ec2InstanceConsoleOutput{
		inst:   inst,
		latest: latest,
	}

	if cl.latest {
		cl.EntryBase = plugin.NewEntry("console-latest.out")
	} else {
		cl.EntryBase = plugin.NewEntry("console.out")
	}

	return cl
}

// TODO: Once https://github.com/puppetlabs/wash/issues/123 is resolved, create
// a CachedConsoleLog method that we can use to generate the content + retrieve
// the console log file's attributes (e.g. like Ctime/Mtime)
func (cl *ec2InstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	request := &ec2Client.GetConsoleOutputInput{
		InstanceId: awsSDK.String(cl.inst.Name()),
	}
	if cl.latest {
		request.Latest = awsSDK.Bool(cl.latest)
	}

	resp, err := cl.inst.client.GetConsoleOutputWithContext(ctx, request)
	if err != nil {
		return nil, err
	}

	content, err := base64.StdEncoding.DecodeString(awsSDK.StringValue(resp.Output))
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

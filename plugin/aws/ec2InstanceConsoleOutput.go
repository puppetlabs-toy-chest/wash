package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"time"

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

func ec2InstanceConsoleOutputTemplate() *ec2InstanceConsoleOutput {
	cl := &ec2InstanceConsoleOutput{
		EntryBase: plugin.NewEntry(),
	}
	cl.SetShortType("console.out")
	return cl
}

func newEC2InstanceConsoleOutput(ctx context.Context, inst *ec2Instance, latest bool) (*ec2InstanceConsoleOutput, error) {
	cl := ec2InstanceConsoleOutputTemplate()
	cl.inst = inst
	cl.latest = latest

	if cl.latest {
		cl.EntryBase = plugin.NewEntry()
		cl.SetName("console-latest.out")
	} else {
		cl.EntryBase = plugin.NewEntry()
		cl.SetName("console.out")
	}
	cl.DisableDefaultCaching()

	output, err := cl.cachedConsoleOutput(ctx)
	if err != nil {
		return nil, err
	}

	cl.
		Attributes().
		SetCtime(output.mtime).
		SetMtime(output.mtime).
		SetAtime(output.mtime).
		SetSize(uint64(len(output.content)))

	return cl, nil
}

type consoleOutput struct {
	mtime   time.Time
	content []byte
}

func (cl *ec2InstanceConsoleOutput) cachedConsoleOutput(ctx context.Context) (consoleOutput, error) {
	output, err := plugin.CachedOp(ctx, "ConsoleOutput", cl, 30*time.Second, func() (interface{}, error) {
		request := &ec2Client.GetConsoleOutputInput{
			InstanceId: awsSDK.String(cl.inst.id),
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

func (cl *ec2InstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	output, err := cl.cachedConsoleOutput(ctx)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(output.content), nil
}

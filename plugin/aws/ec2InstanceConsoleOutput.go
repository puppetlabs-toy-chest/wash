package aws

import (
	"bytes"
	"context"

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
	cl.TurnOffCaching()

	return cl
}

func (cl *ec2InstanceConsoleOutput) Attr(ctx context.Context) (plugin.Attributes, error) {
	output, err := cl.inst.cachedConsoleOutput(ctx, cl.latest)
	if err != nil {
		return plugin.Attributes{}, err
	}

	cl.Ctime = output.mtime
	attr, _ := cl.EntryBase.Attr(ctx)
	attr.Size = uint64(len(output.content))

	return attr, nil
}

func (cl *ec2InstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	output, err := cl.inst.cachedConsoleOutput(ctx, cl.latest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(output.content), nil
}

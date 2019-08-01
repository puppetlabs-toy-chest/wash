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

func newEC2InstanceConsoleOutput(ctx context.Context, inst *ec2Instance, latest bool) (*ec2InstanceConsoleOutput, error) {
	cl := &ec2InstanceConsoleOutput{}
	cl.inst = inst
	cl.latest = latest

	if cl.latest {
		cl.EntryBase = plugin.NewEntry("console-latest.out")
	} else {
		cl.EntryBase = plugin.NewEntry("console.out")
	}

	output, err := cl.inst.cachedConsoleOutput(ctx, cl.latest)
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

func (cl *ec2InstanceConsoleOutput) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cl, "console.out").SetEntryType("ec2InstanceConsoleOutput")
}

func (cl *ec2InstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	output, err := cl.inst.cachedConsoleOutput(ctx, cl.latest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(output.content), nil
}

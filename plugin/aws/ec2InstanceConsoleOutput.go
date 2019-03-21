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
	cl.DisableDefaultCaching()

	return cl
}

func (cl *ec2InstanceConsoleOutput) Attr(ctx context.Context) (plugin.Attributes, error) {
	output, err := cl.inst.cachedConsoleOutput(ctx, cl.latest)
	if err != nil {
		return plugin.Attributes{}, err
	}

	// We can't use cl.EntryBase.Attr() here because that depends on the Ctime
	// field being set. Setting Ctime would introduce a race condition, so manually
	// create the Attributes struct
	return plugin.Attributes{
		Ctime: output.mtime,
		Mtime: output.mtime,
		Atime: output.mtime,
		Size:  uint64(len(output.content)),
	}, nil
}

func (cl *ec2InstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	output, err := cl.inst.cachedConsoleOutput(ctx, cl.latest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(output.content), nil
}

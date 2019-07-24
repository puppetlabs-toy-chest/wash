package gcp

import (
	"context"
	"strings"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	compute "google.golang.org/api/compute/v1"
)

type computeInstanceConsoleOutput struct {
	plugin.EntryBase
	instance *compute.Instance
	computeProjectService
}

func newComputeInstanceConsoleOutput(c computeProjectService, inst *compute.Instance) *computeInstanceConsoleOutput {
	return &computeInstanceConsoleOutput{
		EntryBase:             plugin.NewEntry("console.out"),
		instance:              inst,
		computeProjectService: c,
	}
}

func (cl *computeInstanceConsoleOutput) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cl, "console.out")
}

func (cl *computeInstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	// Zone is given as a URL on the Instance type.
	zoneSlice := strings.Split(cl.instance.Zone, "/")
	zone := zoneSlice[len(zoneSlice)-1]

	activity.Record(ctx,
		"Getting output for instance %v in project %v, zone %v", cl.instance.Name, cl.projectID, zone)
	outputCall := cl.Instances.GetSerialPortOutput(cl.projectID, zone, cl.instance.Name)
	outputResp, err := outputCall.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return strings.NewReader(outputResp.Contents), nil
}

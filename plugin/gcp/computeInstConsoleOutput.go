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
	service  computeProjectService
}

func newComputeInstanceConsoleOutput(inst *compute.Instance, c computeProjectService) *computeInstanceConsoleOutput {
	return &computeInstanceConsoleOutput{
		EntryBase: plugin.NewEntry("console.out"),
		instance:  inst,
		service:   c,
	}
}

func (cl *computeInstanceConsoleOutput) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cl, "console.out").IsSingleton()
}

func (cl *computeInstanceConsoleOutput) Open(ctx context.Context) (plugin.SizedReader, error) {
	zone := getZone(cl.instance)
	activity.Record(ctx,
		"Getting output for instance %v in project %v, zone %v", cl.instance.Name, cl.service.projectID, zone)
	outputCall := cl.service.Instances.GetSerialPortOutput(cl.service.projectID, zone, cl.instance.Name)
	outputResp, err := outputCall.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return strings.NewReader(outputResp.Contents), nil
}

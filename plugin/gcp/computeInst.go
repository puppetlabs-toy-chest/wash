package gcp

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	compute "google.golang.org/api/compute/v1"
)

type computeInstance struct {
	plugin.EntryBase
	instance *compute.Instance
	computeProjectService
}

func newComputeInstance(inst *compute.Instance, c computeProjectService) *computeInstance {
	comp := &computeInstance{
		EntryBase:             plugin.NewEntry(inst.Name),
		instance:              inst,
		computeProjectService: c,
	}
	comp.Attributes().SetMeta(inst)
	return comp
}

func (c *computeInstance) List(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{newComputeInstanceConsoleOutput(c.computeProjectService, c.instance)}, nil
}

func (c *computeInstance) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(c, "instance")
}

func (c *computeInstance) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{}
}

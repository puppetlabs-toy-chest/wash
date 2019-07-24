package gcp

import (
	"context"
	"net/http"

	"github.com/puppetlabs/wash/plugin"
	compute "google.golang.org/api/compute/v1"
)

type computeProjectService struct {
	*compute.Service
	projectID string
}

// TODO: re-use credentials we got during init
type computeDir struct {
	plugin.EntryBase
	computeProjectService
}

const computeScope = compute.CloudPlatformScope

func newComputeDir(client *http.Client, projID string) (*computeDir, error) {
	svc, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	return &computeDir{
		EntryBase:             plugin.NewEntry("compute"),
		computeProjectService: computeProjectService{Service: svc, projectID: projID},
	}, nil
}

// List all services as dirs.
func (c *computeDir) List(ctx context.Context) ([]plugin.Entry, error) {
	var entries []plugin.Entry
	instReq := c.Instances.AggregatedList(c.projectID)
	if err := instReq.Pages(ctx, func(instancePage *compute.InstanceAggregatedList) error {
		for _, zone := range instancePage.Items {
			for _, instance := range zone.Instances {
				inst := newComputeInstance(instance, c.computeProjectService)
				entries = append(entries, inst)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *computeDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(c, "compute").IsSingleton()
}

func (c *computeDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{(&computeInstance{}).Schema()}
}

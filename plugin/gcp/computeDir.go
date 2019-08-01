package gcp

import (
	"context"
	"net/http"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

type computeProjectService struct {
	*compute.Service
	projectID string
}

type computeDir struct {
	plugin.EntryBase
	service computeProjectService
}

const computeScope = compute.CloudPlatformScope

func newComputeDir(client *http.Client, projID string) (*computeDir, error) {
	svc, err := compute.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &computeDir{
		EntryBase: plugin.NewEntry("compute"),
		service:   computeProjectService{Service: svc, projectID: projID},
	}, nil
}

// List all services as dirs.
func (c *computeDir) List(ctx context.Context) ([]plugin.Entry, error) {
	var entries []plugin.Entry
	instReq := c.service.Instances.AggregatedList(c.service.projectID)
	err := instReq.Pages(ctx, func(instancePage *compute.InstanceAggregatedList) error {
		for _, zone := range instancePage.Items {
			for _, instance := range zone.Instances {
				inst := newComputeInstance(instance, c.service)
				entries = append(entries, inst)
			}
		}
		return nil
	})
	return entries, err
}

func (c *computeDir) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	// Return project metadata specific to Google Compute.
	proj, err := c.service.Projects.Get(c.service.projectID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return plugin.ToJSONObject(proj), nil
}

func (c *computeDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(c, "compute").IsSingleton()
}

func (c *computeDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{(&computeInstance{}).Schema()}
}

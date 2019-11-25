package gcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
)

type cloudRunProjectAPIService struct {
	*run.APIService
	projectID string
	// We need to pass this around to access cloud function logs
	client *http.Client
}

type cloudRunDir struct {
	plugin.EntryBase
	apiService cloudRunProjectAPIService
}

func newCloudRunDir(ctx context.Context, client *http.Client, projID string) (*cloudRunDir, error) {
	svc, err := run.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	cr := &cloudRunDir{
		EntryBase:  plugin.NewEntry("cloud_run"),
		apiService: cloudRunProjectAPIService{APIService: svc, projectID: projID, client: client},
	}
	if _, err := plugin.List(ctx, cr); err != nil {
		cr.MarkInaccessible(ctx, err)
	}
	return cr, nil
}

func (cr *cloudRunDir) List(ctx context.Context) ([]plugin.Entry, error) {
	var entries []plugin.Entry
	servicesReq := cr.apiService.Projects.Locations.Services.List(fmt.Sprintf("projects/%s/locations/-", cr.apiService.projectID))
	resp, err := servicesReq.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	for _, service := range resp.Items {
		entries = append(entries, newCloudRunService(service, cr.apiService))
	}
	return entries, err
}

func (cr *cloudRunDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cr, "cloud_run").
		IsSingleton()
}

func (cr *cloudRunDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&cloudRunService{}).Schema(),
	}
}

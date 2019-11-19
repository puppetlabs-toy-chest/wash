package gcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/option"
)

type cloudFunctionsProjectService struct {
	*cloudfunctions.Service
	projectID string
	// We need to pass this around to access cloud function logs
	client *http.Client
}

type cloudFunctionsDir struct {
	plugin.EntryBase
	service cloudFunctionsProjectService
}

func newCloudFunctionsDir(ctx context.Context, client *http.Client, projID string) (*cloudFunctionsDir, error) {
	svc, err := cloudfunctions.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	cf := &cloudFunctionsDir{
		EntryBase: plugin.NewEntry("cloud_functions"),
		service:   cloudFunctionsProjectService{Service: svc, projectID: projID, client: client},
	}
	if _, err := plugin.List(ctx, cf); err != nil {
		cf.MarkInaccessible(ctx, err)
	}
	return cf, nil
}

func (cf *cloudFunctionsDir) List(ctx context.Context) ([]plugin.Entry, error) {
	var entries []plugin.Entry
	functionsReq := cf.service.Projects.Locations.Functions.List(fmt.Sprintf("projects/%v/locations/-", cf.service.projectID))
	err := functionsReq.Pages(ctx, func(resp *cloudfunctions.ListFunctionsResponse) error {
		for _, function := range resp.Functions {
			entries = append(entries, newCloudFunction(function, cf.service))
		}
		return nil
	})
	return entries, err
}

func (cf *cloudFunctionsDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cf, "cloud_functions").
		IsSingleton()
}

func (cf *cloudFunctionsDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&cloudFunction{}).Schema(),
	}
}

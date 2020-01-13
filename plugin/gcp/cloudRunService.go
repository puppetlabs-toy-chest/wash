package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/run/v1"
)

type cloudRunService struct {
	plugin.EntryBase
	apiService cloudRunProjectAPIService
	region     string
}

func newCloudRunService(service *run.Service, apiService cloudRunProjectAPIService) *cloudRunService {
	crs := &cloudRunService{
		EntryBase:  plugin.NewEntry(service.Metadata.Name),
		apiService: apiService,
		region:     service.Metadata.Labels["cloud.googleapis.com/location"],
	}
	crtime, err := time.Parse(time.RFC3339, service.Metadata.CreationTimestamp)
	if err != nil {
		panic(fmt.Sprintf("Timestamp for %v was not expected format RFC3339: %v", crs, service.Metadata.CreationTimestamp))
	}
	crs.
		SetPartialMetadata(service).
		Attributes().
		SetCrtime(crtime)
	return crs
}

func (crs *cloudRunService) List(ctx context.Context) ([]plugin.Entry, error) {
	crsl, err := newCloudRunServiceLog(ctx, crs.apiService, crs.region, crs.Name())
	if err != nil {
		return nil, err
	}
	return []plugin.Entry{crsl}, nil
}

func (crs *cloudRunService) Delete(ctx context.Context) (bool, error) {
	fullResourceName := fmt.Sprintf(
		"projects/%s/locations/%s/services/%s",
		crs.apiService.projectID,
		crs.region,
		crs.Name(),
	)
	_, err := crs.apiService.Projects.Locations.Services.Delete(fullResourceName).Context(ctx).Do()
	return true, err
}
func (crs *cloudRunService) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(crs, "service").
		SetPartialMetadataSchema(run.Service{})
}

func (crs *cloudRunService) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&cloudRunServiceLog{}).Schema(),
	}
}

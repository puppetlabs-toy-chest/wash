package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/cloudfunctions/v1"
)

type cloudFunction struct {
	plugin.EntryBase
	service cloudFunctionsProjectService
	region  string
}

func newCloudFunction(function *cloudfunctions.CloudFunction, service cloudFunctionsProjectService) *cloudFunction {
	// functionPath is formatted as projects/<project_id>/locations/<region>/functions/<function_name>
	functionPath := function.Name
	segments := strings.Split(functionPath, "/")
	region, functionName := segments[3], segments[5]
	cf := &cloudFunction{
		EntryBase: plugin.NewEntry(functionName),
		service:   service,
		region:    region,
	}
	mtime, err := time.Parse(time.RFC3339, function.UpdateTime)
	if err != nil {
		panic(fmt.Sprintf("Timestamp for %v was not expected format RFC3339: %v", cf, function.UpdateTime))
	}
	cf.
		Attributes().
		SetMtime(mtime).
		SetMeta(function)
	return cf
}

func (cf *cloudFunction) List(ctx context.Context) ([]plugin.Entry, error) {
	cfl, err := newCloudFunctionLog(ctx, cf.service, cf.region, cf.Name())
	if err != nil {
		return nil, err
	}
	return []plugin.Entry{cfl}, nil
}

func (cf *cloudFunction) Delete(ctx context.Context) (bool, error) {
	_, err := cf.service.Projects.Locations.Functions.Delete(cf.Name()).Context(ctx).Do()
	return false, err
}

func (cf *cloudFunction) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cf, "cloud_function").
		SetMetaAttributeSchema(cloudfunctions.CloudFunction{})
}

func (cf *cloudFunction) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&cloudFunctionLog{}).Schema(),
	}
}

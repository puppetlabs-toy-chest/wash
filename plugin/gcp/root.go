// Package gcp presents a filesystem hierarchy for Google Cloud Platform resources.
package gcp

import (
	"context"
	"net/http"
	"time"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"golang.org/x/oauth2/google"
	crm "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

// Root of the GCP plugin
type Root struct {
	plugin.EntryBase
	oauthClient *http.Client
}

// serviceScopes lists all scopes used by this module.
var serviceScopes = []string{crm.CloudPlatformScope, computeScope}

// Init for root
func (r *Root) Init(cfg map[string]interface{}) error {
	r.EntryBase = plugin.NewEntry("gcp")
	r.SetTTLOf(plugin.ListOp, 1*time.Minute)

	// We use the auto-generated SDK because it's the only one that allows us to list
	// projects for the current credentials.
	oauthClient, err := google.DefaultClient(context.Background(), serviceScopes...)
	r.oauthClient = oauthClient
	return err
}

// ChildSchemas returns the root's child schema
func (r *Root) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&project{}).Schema(),
	}
}

// Schema returns the root's schema
func (r *Root) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(r, "gcp").IsSingleton()
}

// List the available GCP projects
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	crmService, err := crm.NewService(context.Background(), option.WithHTTPClient(r.oauthClient))
	if err != nil {
		return nil, err
	}

	activity.Record(ctx, "Loading projects for %v", crmService.BasePath)

	listResponse, err := crmService.Projects.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	projects := make([]plugin.Entry, len(listResponse.Projects))
	for i, proj := range listResponse.Projects {
		projects[i] = newProject(proj, r.oauthClient)
	}
	return projects, nil
}

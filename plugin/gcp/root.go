// Package gcp presents a filesystem hierarchy for Google Cloud Platform resources.
//
// It follows https://cloud.google.com/docs/authentication/production to find your credentials:
// - it will try `GOOGLE_APPLICATION_CREDENTIALS` as a service account file
// - use your credentials in `$HOME/.config/gcloud/application_default_credentials.json`
//
// The simplest way to set this up is with
//     gcloud init
//     gcloud auth application-default login
package gcp

import (
	"context"
	"net/http"
	"time"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"golang.org/x/oauth2/google"
	crm "google.golang.org/api/cloudresourcemanager/v1"
)

// Root of the GCP plugin
type Root struct {
	plugin.EntryBase
	oauthClient *http.Client
}

// Init for root
func (r *Root) Init(cfg map[string]interface{}) error {
	r.EntryBase = plugin.NewEntry("gcp")
	r.SetTTLOf(plugin.ListOp, 1*time.Minute)

	// We use the auto-generated SDK because it's the only one that allows us to list
	// projects for the current credentials.
	oauthClient, err := google.DefaultClient(context.Background(), crm.CloudPlatformScope)
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
	crmService, err := crm.New(r.oauthClient)
	if err != nil {
		return nil, err
	}
	lister := crm.NewProjectsService(crmService).List()

	activity.Record(ctx, "Loading projects for %v", crmService.BasePath)

	listResponse, err := lister.Do()
	if err != nil {
		return nil, err
	}

	projects := make([]plugin.Entry, len(listResponse.Projects))
	for i, proj := range listResponse.Projects {
		projects[i] = newProject(proj)
	}
	return projects, nil
}

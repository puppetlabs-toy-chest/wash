// Package gcp presents a filesystem hierarchy for Google Cloud Platform resources.
package gcp

import (
	"context"
	"fmt"
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
	projects    map[string]struct{}
}

// serviceScopes lists all scopes used by this module.
var serviceScopes = []string{crm.CloudPlatformScope, computeScope, storageScope}

// Init for root
func (r *Root) Init(cfg map[string]interface{}) error {
	r.EntryBase = plugin.NewEntry("gcp")
	r.SetTTLOf(plugin.ListOp, 1*time.Minute)

	// We use the auto-generated SDK because it's the only one that allows us to list
	// projects for the current credentials.
	oauthClient, err := google.DefaultClient(context.Background(), serviceScopes...)
	r.oauthClient = oauthClient

	if projsI, ok := cfg["projects"]; ok {
		projs, ok := projsI.([]interface{})
		if !ok {
			return fmt.Errorf("gcp.projects config must be an array of strings, not %s", projsI)
		}
		r.projects = make(map[string]struct{})
		for _, elem := range projs {
			proj, ok := elem.(string)
			if !ok {
				return fmt.Errorf("gcp.projects config must be an array of strings, not %s", projs)
			}
			r.projects[proj] = struct{}{}
		}
	}

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

	projects := make([]plugin.Entry, 0, len(listResponse.Projects))
	for _, proj := range listResponse.Projects {
		if _, ok := r.projects[proj.Name]; len(r.projects) > 0 && !ok {
			if _, ok := r.projects[proj.ProjectId]; !ok {
				// If a list of enabled projects is provided and both name and project ID are not in it,
				// omit this project.
				continue
			}
		}
		projects = append(projects, newProject(proj, r.oauthClient))
	}
	return projects, nil
}

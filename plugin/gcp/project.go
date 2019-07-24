package gcp

import (
	"context"
	"net/http"

	"github.com/puppetlabs/wash/plugin"
	crm "google.golang.org/api/cloudresourcemanager/v1"
)

type project struct {
	plugin.EntryBase
	client *http.Client
	id     string
}

// NewProject creates a new project with a collection of service clients.
func newProject(p *crm.Project, client *http.Client) *project {
	name := p.Name
	if name == "" {
		name = p.ProjectId
	}
	proj := &project{EntryBase: plugin.NewEntry(name), client: client, id: p.ProjectId}
	proj.Attributes().SetMeta(p)
	return proj
}

// List all services as dirs.
func (p *project) List(ctx context.Context) ([]plugin.Entry, error) {
	comp, err := newComputeDir(p.client, p.id)
	if err != nil {
		return nil, err
	}
	return []plugin.Entry{comp}, nil
}

func (p *project) Schema() *plugin.EntrySchema {
	schema := plugin.NewEntrySchema(p, "project")
	schema.SetMetaAttributeSchema(crm.Project{})
	return schema
}

func (p *project) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&computeDir{}).Schema(),
	}
}

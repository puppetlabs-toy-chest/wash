package gcp

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	crm "google.golang.org/api/cloudresourcemanager/v1"
)

type project struct {
	plugin.EntryBase
}

// NewProject creates a new project with a collection of service clients.
func newProject(p *crm.Project) *project {
	name := p.Name
	if name == "" {
		name = p.ProjectId
	}
	proj := &project{plugin.NewEntry(name)}
	proj.Attributes().SetMeta(p)
	return proj
}

// List all services as dirs.
func (p *project) List(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{}, nil
}

func (p *project) Schema() *plugin.EntrySchema {
	schema := plugin.NewEntrySchema(p, "project")
	schema.SetMetaAttributeSchema(crm.Project{})
	return schema
}

func (p *project) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{}
}

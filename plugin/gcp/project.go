package gcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/puppetlabs/wash/plugin"
	crm "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
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
	var wg sync.WaitGroup
	var mux sync.Mutex
	var errs []string
	var children []plugin.Entry

	save := func(ent plugin.Entry, err error) {
		defer wg.Done()
		mux.Lock()
		defer mux.Unlock()
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			children = append(children, ent)
		}
	}

	go func() { save(newComputeDir(ctx, p.client, p.id)) }()
	go func() { save(newStorageDir(ctx, p.client, p.id)) }()
	go func() { save(newFirestoreDir(ctx, p.id)) }()
	go func() { save(newPubsubDir(ctx, p.id)) }()
	go func() { save(newCloudFunctionsDir(ctx, p.client, p.id)) }()
	go func() { save(newCloudRunDir(ctx, p.client, p.id)) }()
	wg.Add(6)
	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf(strings.Join(errs, ", "))
	}
	return children, nil
}

func (p *project) Delete(ctx context.Context) (bool, error) {
	crmService, err := crm.NewService(context.Background(), option.WithHTTPClient(p.client))
	if err != nil {
		return false, err
	}
	_, err = crmService.Projects.Delete(p.id).Context(ctx).Do()
	return true, err
}

func (p *project) Schema() *plugin.EntrySchema {
	schema := plugin.NewEntrySchema(p, "project")
	schema.SetMetaAttributeSchema(crm.Project{})
	schema.SetDescription(projectDescription)
	return schema
}

func (p *project) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&computeDir{}).Schema(),
		(&storageDir{}).Schema(),
		(&firestoreDir{}).Schema(),
		(&pubsubDir{}).Schema(),
		(&cloudFunctionsDir{}).Schema(),
		(&cloudRunDir{}).Schema(),
	}
}

const projectDescription = `
This is a GCP project.
`

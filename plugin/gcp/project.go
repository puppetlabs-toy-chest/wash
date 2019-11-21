package gcp

import (
	"context"
	"net/http"

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
	comp, err := newComputeDir(ctx, p.client, p.id)
	if err != nil {
		return nil, err
	}

	stor, err := newStorageDir(ctx, p.client, p.id)
	if err != nil {
		return nil, err
	}

	firestore, err := newFirestoreDir(ctx, p.id)
	if err != nil {
		return nil, err
	}

	pubsub, err := newPubsubDir(ctx, p.id)
	if err != nil {
		return nil, err
	}

	cloudFunctions, err := newCloudFunctionsDir(ctx, p.client, p.id)
	if err != nil {
		return nil, err
	}

	children := []plugin.Entry{
		comp,
		stor,
		firestore,
		pubsub,
		cloudFunctions,
	}
	return children, nil
}

func (p *project) Delete(ctx context.Context) (bool, error) {
	crmService, err := crm.NewService(context.Background(), option.WithHTTPClient(p.client))
	if err != nil {
		return false, err
	}
	_, err = crmService.Projects.Delete(p.id).Do()
	return true, err
}

func (p *project) Schema() *plugin.EntrySchema {
	schema := plugin.NewEntrySchema(p, "project")
	schema.SetMetaAttributeSchema(crm.Project{})
	return schema
}

func (p *project) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&computeDir{}).Schema(),
		(&storageDir{}).Schema(),
		(&firestoreDir{}).Schema(),
		(&pubsubDir{}).Schema(),
		(&cloudFunctionsDir{}).Schema(),
	}
}

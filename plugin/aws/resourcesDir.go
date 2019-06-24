package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/puppetlabs/wash/plugin"
)

// resourcesDir represents the <profile>/resources directory
type resourcesDir struct {
	plugin.EntryBase
	session   *session.Session
	resources []plugin.Entry
}

func newResourcesDir(session *session.Session) *resourcesDir {
	resourcesDir := &resourcesDir{
		EntryBase: plugin.NewEntry("resources"),
	}
	resourcesDir.DisableDefaultCaching()
	resourcesDir.session = session

	resourcesDir.resources = []plugin.Entry{
		newS3Dir(resourcesDir.session),
		newEC2Dir(resourcesDir.session),
	}

	return resourcesDir
}

func (r *resourcesDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(r, "resources").IsSingleton()
}

func (r *resourcesDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&s3Dir{}).Schema(),
		(&ec2Dir{}).Schema(),
	}
}

// List lists the available AWS resources
func (r *resourcesDir) List(ctx context.Context) ([]plugin.Entry, error) {
	return r.resources, nil
}

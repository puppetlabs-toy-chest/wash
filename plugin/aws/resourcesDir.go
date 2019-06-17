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

func resourcesDirBase() *resourcesDir {
	resourcesDir := &resourcesDir{
		EntryBase: plugin.NewEntryBase(),
	}
	resourcesDir.
		SetName("resources").
		IsSingleton().
		DisableDefaultCaching()
	return resourcesDir
}

func newResourcesDir(session *session.Session) *resourcesDir {
	resourcesDir := resourcesDirBase()
	resourcesDir.session = session

	resourcesDir.resources = []plugin.Entry{
		newS3Dir(resourcesDir.session),
		newEC2Dir(resourcesDir.session),
	}

	return resourcesDir
}

func (r *resourcesDir) ChildSchemas() []*plugin.EntrySchema {
	return plugin.ChildSchemas(s3DirBase(), ec2DirBase())
}

// List lists the available AWS resources
func (r *resourcesDir) List(ctx context.Context) ([]plugin.Entry, error) {
	return r.resources, nil
}

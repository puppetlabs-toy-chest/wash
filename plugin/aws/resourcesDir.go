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

func newResourcesDir(parent plugin.Entry, session *session.Session) *resourcesDir {
	resourcesDir := &resourcesDir{
		EntryBase: parent.NewEntry("resources"),
		session:   session,
	}
	resourcesDir.DisableDefaultCaching()

	resourcesDir.resources = []plugin.Entry{
		newS3Dir(parent, resourcesDir.session),
		newEC2Dir(parent, resourcesDir.session),
	}

	return resourcesDir
}

// List lists the available AWS resources
func (r *resourcesDir) List(ctx context.Context) ([]plugin.Entry, error) {
	return r.resources, nil
}

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// profile represents an AWS profile
type profile struct {
	plugin.EntryBase
	session      *session.Session
	resourcesDir []plugin.Entry
}

func newProfile(ctx context.Context, name string) (*profile, error) {
	profile := &profile{
		EntryBase: plugin.NewEntry(name),
	}
	profile.DisableDefaultCaching()

	activity.Record(ctx, "Creating a new AWS session for the %v profile", name)

	// profile-specific stdin prompt
	tokenProvider := func() (string, error) {
		return plugin.Prompt(fmt.Sprintf("Assume ROLE MFA token code for %v", name))
	}

	// Create the session. SharedConfigEnable tells AWS to load the profile
	// config from the ~/.aws/credentials and ~/.aws/config files
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:                 name,
		AssumeRoleTokenProvider: tokenProvider,
		// TODO: make this configurable. Different IAM configs may allow different durations.
		// Use the minimum IAM limit of 1 hour.
		AssumeRoleTokenDuration: 1 * time.Hour,
		SharedConfigState:       session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	if cacheProvider, err := newFileCacheProvider(ctx, name, sess.Config.Credentials); err == nil {
		sess.Config.Credentials = credentials.NewCredentials(&cacheProvider)
	} else {
		activity.Record(ctx, "Unable to use cached credentials for %v profile: %v", name, err)
	}

	// Force retrieving credentials now to expose errors early.
	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, fmt.Errorf("Unable to get credentials for %v: %v", name, err)
	}

	profile.session = sess
	profile.resourcesDir = []plugin.Entry{newResourcesDir(sess)}

	return profile, nil
}

func (p *profile) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(p, "profile")
}

func (p *profile) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&resourcesDir{}).Schema(),
	}
}

// List lists the resources directory
func (p *profile) List(ctx context.Context) ([]plugin.Entry, error) {
	return p.resourcesDir, nil
}

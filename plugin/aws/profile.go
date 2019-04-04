package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

// profile represents an AWS profile
type profile struct {
	plugin.EntryBase
	session      *session.Session
	resourcesDir []plugin.Entry
}

func newProfile(ctx context.Context, parent plugin.Entry, name string) (*profile, error) {
	profile := profile{EntryBase: parent.NewEntry(name)}
	profile.DisableDefaultCaching()

	journal.Record(ctx, "Creating a new AWS session for the %v profile", name)

	// Create the session. SharedConfigEnable tells AWS to load the profile
	// config from the ~/.aws/credentials and ~/.aws/config files
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:                 name,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
		SharedConfigState:       session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	if cacheProvider, err := newFileCacheProvider(ctx, name, sess.Config.Credentials); err == nil {
		sess.Config.Credentials = credentials.NewCredentials(&cacheProvider)
	} else {
		journal.Record(ctx, "Unable to use cached credentials for %v profile: %v", name, err)
	}

	// Force retrieving credentials now to expose errors early.
	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, fmt.Errorf("Unable to get credentials for %v: %v", name, err)
	}

	profile.session = sess
	profile.resourcesDir = []plugin.Entry{newResourcesDir(&profile, sess)}

	return &profile, nil
}

// List lists the resources directory
func (p *profile) List(ctx context.Context) ([]plugin.Entry, error) {
	return p.resourcesDir, nil
}

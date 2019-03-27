package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

// profile represents an AWS profile
type profile struct {
	plugin.EntryBase
	resourcesDir []plugin.Entry
}

func newProfile(ctx context.Context, name string) (*profile, error) {
	profile := profile{EntryBase: plugin.NewEntry(name)}
	profile.DisableDefaultCaching()

	journal.Record(ctx, "Creating a new AWS session for the %v profile", name)

	// Create the session. SharedConfigEnable tells AWS to load the profile
	// config from the ~/.aws/credentials and ~/.aws/config files
	session, err := session.NewSessionWithOptions(session.Options{
		Profile:                 profile.Name(),
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
		SharedConfigState:       session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	profile.resourcesDir = []plugin.Entry{newResourcesDir(session)}

	return &profile, nil
}

// List lists the resources directory
func (p *profile) List(ctx context.Context) ([]plugin.Entry, error) {
	return p.resourcesDir, nil
}

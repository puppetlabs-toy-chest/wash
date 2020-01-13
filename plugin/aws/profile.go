package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	stsClient "github.com/aws/aws-sdk-go/service/sts"
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
		AssumeRoleDuration: 1 * time.Hour,
		SharedConfigState:  session.SharedConfigEnable,
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
	return plugin.
		NewEntrySchema(p, "profile").
		SetDescription(profileDescription)
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

type profileMetadata struct {
	AccessKeyID  string
	Account      *string
	Arn          *string
	Name         string
	ProviderName string
	Region       *string
	UserID       *string
}

// Metadata for AWS Profile
func (p *profile) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	cred, err := p.session.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	stsC := stsClient.New(p.session)
	request := &stsClient.GetCallerIdentityInput{}
	resp, err := stsC.GetCallerIdentityWithContext(ctx, request)

	if err != nil {
		return nil, err
	}

	var metadata profileMetadata
	metadata.AccessKeyID = cred.AccessKeyID
	metadata.Account = resp.Account
	metadata.Arn = resp.Arn
	metadata.Name = p.Name()
	metadata.ProviderName = cred.ProviderName
	metadata.Region = p.session.Config.Region
	metadata.UserID = resp.UserId

	return plugin.ToJSONObject(metadata), nil
}

const profileDescription = `
This is an AWS profile.
`

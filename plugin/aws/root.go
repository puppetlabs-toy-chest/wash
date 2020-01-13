// Package aws presents a filesystem hierarchy for AWS resources.
//
// It uses the AWS_SHARED_CREDENTIALS_FILE environment variable or
// $HOME/.aws/credentials to configure AWS access.
package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"gopkg.in/go-ini/ini.v1"
)

// Root of the AWS plugin
type Root struct {
	plugin.EntryBase
	profs map[string]struct{}
}

func awsCredentialsFile() (string, error) {
	if filename := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); len(filename) != 0 {
		return filename, nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine the location of the AWS credentials file: %v", err)
	}

	return filepath.Join(homedir, ".aws", "credentials"), nil
}

func awsConfigFile() (string, error) {
	if filename := os.Getenv("AWS_CONFIG_FILE"); len(filename) != 0 {
		return filename, nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine the location of the AWS config file: %v", err)
	}

	return filepath.Join(homedir, ".aws", "config"), nil
}

func exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

// Init for root
func (r *Root) Init(cfg map[string]interface{}) error {
	r.EntryBase = plugin.NewEntry("aws")
	r.SetTTLOf(plugin.ListOp, 1*time.Minute)

	if profsI, ok := cfg["profiles"]; ok {
		profs, ok := profsI.([]interface{})
		if !ok {
			return fmt.Errorf("aws.profiles config must be an array of strings, not %s", profsI)
		}

		r.profs = make(map[string]struct{})
		for _, elem := range profs {
			prof, ok := elem.(string)
			if !ok {
				return fmt.Errorf("aws.profiles config must be an array of strings, not %s", profs)
			}
			r.profs[prof] = struct{}{}
		}
	}

	// Force authorizing profiles on startup
	_, err := r.List(context.Background())
	return err
}

// ChildSchemas returns the root's child schema
func (r *Root) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&profile{}).Schema(),
	}
}

// Schema returns the root's schema
func (r *Root) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(r, "aws").
		SetDescription(rootDescription).
		IsSingleton()
}

// List the available AWS profiles
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	awsCredentials, err := awsCredentialsFile()
	if err != nil {
		return nil, err
	}

	awsCredentialsExists, err := exists(awsCredentials)
	if err != nil {
		return nil, err
	}

	awsConfig, err := awsConfigFile()
	if err != nil {
		return nil, err
	}

	awsConfigExists, err := exists(awsConfig)
	if err != nil {
		return nil, err
	}

	if !awsCredentialsExists && !awsConfigExists {
		return nil, fmt.Errorf("could not load any profiles: the %v and %v files do not exist", awsCredentials, awsConfig)
	}

	var loadedFiles string
	if awsCredentialsExists && awsConfigExists {
		loadedFiles = fmt.Sprintf("%v and %v", awsCredentials, awsConfig)
	} else if !awsCredentialsExists {
		loadedFiles = fmt.Sprintf("%v", awsConfig)
	} else {
		loadedFiles = fmt.Sprintf("%v", awsCredentials)
	}
	activity.Record(ctx, "Loading profiles from %v", loadedFiles)

	names := make(map[string]struct{})

	if awsCredentialsExists {
		cred, err := ini.Load(awsCredentials)
		if err != nil {
			return nil, fmt.Errorf("failed to read %v: %v", awsCredentials, err)
		}
		for _, section := range cred.Sections() {
			names[section.Name()] = struct{}{}
		}
	}
	if awsConfigExists {
		config, err := ini.Load(awsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read %v: %v", awsConfig, err)
		}
		for _, section := range config.Sections() {
			// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
			// Named profiles in config begin with 'profile '. Trim that so config and credentials
			// entries match up.
			names[strings.TrimPrefix(section.Name(), "profile ")] = struct{}{}
		}
	}

	profiles := make([]plugin.Entry, 0, len(names))
	for name := range names {
		if name == "DEFAULT" {
			continue
		}

		if _, ok := r.profs[name]; len(r.profs) > 0 && !ok {
			continue
		}

		profile, err := newProfile(ctx, name)
		if err != nil {
			activity.Warnf(ctx, err.Error())
			continue
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

const rootDescription = `
This is the AWS plugin root. The AWS plugin reads the AWS_SHARED_CREDENTIALS_FILE
environment variable or $HOME/.aws/credentials and AWS_CONFIG_FILE environment
variable or $HOME/.aws/config to find profiles and configure the SDK. See
https://docs.aws.amazon.com/sdk-for-php/v3/developer-guide/guide_credentials_profiles.html
for more details on how to setup AWS profiles.

You can limit the loaded profiles by adding

aws:
  profiles: [profile_1, profile_2]

to Wash’s config file.

The AWS plugin currently supports EC2 and S3. IAM roles are supported when configured
as described here. Note that currently region will also need to be specified with the
profile.

If using MFA, Wash will prompt for it on standard input. Credentials are valid for 1 hour.
They are cached under wash/aws-credentials in your user cache directory so they can be
re-used across server restarts. Wash may have to re-prompt for a new MFA token in response
to navigating the Wash environment to authorize a new session.
`

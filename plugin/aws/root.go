// Package aws presents a filesystem hierarchy for AWS resources.
//
// It uses the AWS_SHARED_CREDENTIALS_FILE environment variable or
// $HOME/.aws/credentials to configure AWS access.
package aws

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	"gopkg.in/go-ini/ini.v1"
)

// Root of the AWS plugin
type Root struct {
	plugin.EntryBase
}

func homeDir() (string, error) {
	curUser, err := user.Current()
	if err != nil {
		return "", err
	}

	if curUser.HomeDir == "" {
		return "", fmt.Errorf("the current user %v does not have a home directory", curUser.Name)
	}
	return curUser.HomeDir, nil
}

func awsCredentialsFile() (string, error) {
	if filename := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); len(filename) != 0 {
		return filename, nil
	}

	homedir, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine the location of the AWS credentials file: %v", err)
	}

	return filepath.Join(homedir, ".aws", "credentials"), nil
}

func awsConfigFile() (string, error) {
	if filename := os.Getenv("AWS_CONFIG_FILE"); len(filename) != 0 {
		return filename, nil
	}

	homedir, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine the location of the AWS config file: %v", err)
	}

	return filepath.Join(homedir, ".aws", "config"), nil
}

func exists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("could not load any profiles: the %v file does not exist", path)
		}

		return err
	}
	return nil
}

// Init for root
func (r *Root) Init() error {
	r.EntryBase = plugin.NewEntry("aws")
	r.SetTTLOf(plugin.List, 1*time.Minute)

	// Force authorizing profiles on startup
	_, err := r.List(context.Background())
	return err
}

// List the available AWS profiles
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	awsCredentials, err := awsCredentialsFile()
	if err != nil {
		return nil, err
	}

	if err := exists(awsCredentials); err != nil {
		return nil, err
	}

	awsConfig, err := awsConfigFile()
	if err != nil {
		return nil, err
	}

	if err := exists(awsConfig); err != nil {
		return nil, err
	}

	journal.Record(ctx, "Loading profiles from %v and %v", awsConfig, awsCredentials)

	cred, err := ini.Load(awsCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to read %v: %v", awsCredentials, err)
	}

	config, err := ini.Load(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read %v: %v", awsConfig, err)
	}

	names := make(map[string]struct{})
	for _, section := range cred.Sections() {
		names[section.Name()] = struct{}{}
	}
	for _, section := range config.Sections() {
		// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
		// Named profiles in config begin with 'profile '. Trim that so config and credentials
		// entries match up.
		names[strings.TrimPrefix(section.Name(), "profile ")] = struct{}{}
	}

	var profiles []plugin.Entry
	for name := range names {
		if name == "DEFAULT" {
			continue
		}

		profile, err := newProfile(ctx, name)
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

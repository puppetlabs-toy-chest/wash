package aws

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	"gopkg.in/go-ini/ini.v1"
)

// Root of the AWS plugin
type Root struct {
	plugin.EntryBase
}

func awsCredentialsFile() (string, error) {
	if filename := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); len(filename) != 0 {
		return filename, nil
	}

	curUser, err := user.Current()
	if err != nil {
		return "", err
	}

	if curUser.HomeDir == "" {
		return "", fmt.Errorf(
			"could not determine the location of the AWS credentials file: the current user %v does not have a home directory",
			curUser.Name,
		)
	}

	return filepath.Join(curUser.HomeDir, ".aws", "credentials"), nil
}

// Init for root
func (r *Root) Init() error {
	r.EntryBase = plugin.NewEntry("aws")
	r.SetTTLOf(plugin.List, 1*time.Minute)

	return nil
}

// List the available AWS profiles
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	awsCredentials, err := awsCredentialsFile()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(awsCredentials); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("could not load any profiles: the %v file does not exist", awsCredentials)
		}

		return nil, err
	}

	journal.Record(ctx, "Loading the profiles from %v", awsCredentials)

	config, err := ini.Load(awsCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to read %v: %v", awsCredentials, err)
	}

	var profiles []plugin.Entry
	for _, section := range config.Sections() {
		name := section.Name()
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

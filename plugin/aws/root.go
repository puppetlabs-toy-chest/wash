package aws

import (
	"context"
	"fmt"
	"os"
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

func awsCredentialsFile() string {
	if filename := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); len(filename) != 0 {
		return filename
	}

	return filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
}

// Init for root
func (r *Root) Init() error {
	r.EntryBase = plugin.NewEntry("aws")
	r.CacheConfig().SetTTLOf(plugin.List, 5*time.Minute)

	return nil
}

// List lists the available AWS profiles
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	awsCredentials := awsCredentialsFile()
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

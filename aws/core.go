package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"github.com/aws/aws-sdk-go/aws/session"

	"gopkg.in/go-ini/ini.v1"
)

type root struct {
	*session.Session
	cache      *datastore.MemCache
	updated    time.Time
	root       string
	resources *resources
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond

func awsCredentialsFile() string {
	if filename := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); len(filename) != 0 {
		return filename
	}

	return filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
}

// ListProfiles lists the available AWS profiles. It reads this information
// from the ~/.aws/credentials file. Unfortunately, aws-sdk-go does not have
// an "AllProfiles" method that we can use while the one provided by awless
// does not report any errors.
//
func ListProfiles() ([]string, error) {
	awsCredentials := awsCredentialsFile()
	if _, err := os.Stat(awsCredentials); err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, err
	}

	config, err := ini.Load(awsCredentials)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %v: %v", awsCredentials, err)
	}

	sections := config.Sections()
	var profiles []string
	for _, section := range sections {
		if name := section.Name(); name != "DEFAULT" {
			profiles = append(profiles, name)
		}
	}

	return profiles, nil
}

// Create a new AWS client.
func Create(name string, context interface{}, cache *datastore.MemCache) (plugin.DirProtocol, error) {
	profile := context.(string)

	// Create the session. SharedConfigEnable tells AWS to load the profile
	// config from the ~/.aws/credentials and ~/.aws/config files
	session, err := session.NewSessionWithOptions(session.Options{
		Profile: profile,
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	cli := &root{session, cache, time.Now(), name, nil}
	cli.resources = newResources(cli)

	return cli, nil
}

// Find the AWS resource.
func (cli *root) Find(ctx context.Context, name string) (plugin.Node, error) {
	if name != "resources" {
		return nil, plugin.ENOENT
	}

	return plugin.NewDir(cli.resources), nil
}

// List all namespaces.
func (cli *root) List(ctx context.Context) ([]plugin.Node, error) {
	return []plugin.Node{plugin.NewDir(cli.resources)}, nil
}

// Name returns the root directory of the client.
func (cli *root) Name() string {
	return cli.root
}

// Attr returns attributes of the named resource.
func (cli *root) Attr(ctx context.Context) (*plugin.Attributes, error) {
	latest := cli.updated
	resourcesAttr, err := cli.resources.Attr(ctx)
	if err != nil {
		return nil, err
	}
	if resourcesAttr.Mtime.After(latest) {
		latest = resourcesAttr.Mtime
	}

	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *root) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

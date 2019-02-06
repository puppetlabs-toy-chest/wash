package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type root struct {
	*client.Client
	cache         *datastore.MemCache
	updated       time.Time
	root          string
	resourcetypes map[string]*resourcetype
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond

// Create a new docker client.
func Create(name string, _ interface{}, cache *datastore.MemCache) (plugin.DirProtocol, error) {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cli := &root{dockerCli, cache, time.Now(), name, nil}
	cli.resourcetypes = newResourceTypes(cli)
	return cli, nil
}

// Find container by ID.
func (cli *root) Find(ctx context.Context, name string) (plugin.Node, error) {
	if rt, ok := cli.resourcetypes[name]; ok {
		log.Debugf("Found resource type %v", rt)
		return plugin.NewDir(rt), nil
	}
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *root) List(ctx context.Context) ([]plugin.Node, error) {
	log.Debugf("Listing %v resource types in %v", len(cli.resourcetypes), cli.Name())
	entries := make([]plugin.Node, 0, len(cli.resourcetypes))
	for _, rt := range cli.resourcetypes {
		entries = append(entries, plugin.NewDir(rt))
	}
	return entries, nil
}

// Name returns the root directory of the client.
func (cli *root) Name() string {
	return cli.root
}

// Attr returns attributes of the named resource.
func (cli *root) Attr(ctx context.Context) (*plugin.Attributes, error) {
	// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
	latest := cli.updated
	for _, v := range cli.resourcetypes {
		attr, err := v.Attr(ctx)
		if err != nil {
			return nil, err
		}
		if attr.Mtime.After(latest) {
			latest = attr.Mtime
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *root) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

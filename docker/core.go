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

// Find the resource type by its ID.
func (cli *root) Find(ctx context.Context, name string) (plugin.Node, error) {
	if rt, ok := cli.resourcetypes[name]; ok {
		log.Debugf("Found resource type %v", rt)
		return plugin.NewDir(rt), nil
	}
	return nil, plugin.ENOENT
}

// List the available resource types as directories
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
	return &plugin.Attributes{Mtime: latest}, nil
}

// Xattr returns a map of extended attributes.
func (cli *root) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

package docker

import (
	"context"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type root struct {
	*client.Client
	*bigcache.BigCache
	mux           sync.Mutex
	reqs          map[string]*datastore.StreamBuffer
	updated       time.Time
	root          string
	resourcetypes map[string]*resourcetype
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond

// Create a new docker client.
func Create(name string, cache *bigcache.BigCache) (plugin.DirProtocol, error) {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	reqs := make(map[string]*datastore.StreamBuffer)
	cli := &root{dockerCli, cache, sync.Mutex{}, reqs, time.Now(), name, nil}
	cli.resourcetypes = newResourceTypes(cli)
	return cli, nil
}

// Find container by ID.
func (cli *root) Find(ctx context.Context, name string) (plugin.Node, error) {
	if rt, ok := cli.resourcetypes[name]; ok {
		log.Debugf("Found resource type %v, %v", name, rt)
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
	for _, v := range cli.reqs {
		if updated := v.LastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *root) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

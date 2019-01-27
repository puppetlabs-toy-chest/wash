package docker

import (
	"bytes"
	"context"
	"encoding/gob"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type root struct {
	*client.Client
	*bigcache.BigCache
	mux     sync.Mutex
	reqs    map[string]*datastore.StreamBuffer
	updated time.Time
	root    string
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const (
	validDuration = 100 * time.Millisecond
	headerLen     = 8
	headerSizeIdx = 4
)

// Create a new docker client.
func Create(name string) (plugin.DirProtocol, error) {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	config := bigcache.DefaultConfig(1 * time.Second)
	config.CleanWindow = 100 * time.Millisecond
	cache, err := bigcache.NewBigCache(config)
	if err != nil {
		return nil, err
	}

	reqs := make(map[string]*datastore.StreamBuffer)
	return &root{dockerCli, cache, sync.Mutex{}, reqs, time.Now(), name}, nil
}

// Find container by ID.
func (cli *root) Find(ctx context.Context, name string) (plugin.Node, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	for _, inst := range containers {
		if inst.ID == name {
			log.Debugf("Found container %v, %v", name, inst)
			return plugin.NewFile(&container{cli, inst.ID}), nil
		}
	}
	log.Debugf("Container %v not found", name)
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *root) List(ctx context.Context) ([]plugin.Node, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	log.Debugf("Listing %v containers in /docker", len(containers))
	keys := make([]plugin.Node, len(containers))
	for i, inst := range containers {
		keys[i] = plugin.NewFile(&container{cli, inst.ID})
	}
	return keys, nil
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

func (cli *root) cachedContainerList(ctx context.Context) ([]types.Container, error) {
	entry, err := cli.Get("ContainerList")
	var containers []types.Container
	if err == nil {
		log.Debugf("Cache hit in /docker")
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&containers)
	} else {
		log.Debugf("Cache miss in /docker")
		containers, err = cli.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(&containers); err != nil {
			return nil, err
		}
		cli.Set("ContainerList", data.Bytes())
		cli.updated = time.Now()
	}
	return containers, err
}

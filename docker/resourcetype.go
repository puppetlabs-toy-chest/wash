package docker

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type resourcetype struct {
	*root
	typename string
}

func newResourceTypes(cli *root) map[string]*resourcetype {
	resourcetypes := make(map[string]*resourcetype)
	for _, name := range []string{"container"} {
		resourcetypes[name] = &resourcetype{cli, name}
	}
	return resourcetypes
}

// Find resource by ID.
func (cli *resourcetype) Find(ctx context.Context, name string) (plugin.Node, error) {
	switch cli.typename {
	case "container":
		containers, err := cli.cachedContainerList(ctx)
		if err != nil {
			return nil, err
		}
		for _, inst := range containers {
			if inst.ID == name {
				log.Debugf("Found container %v", inst)
				return plugin.NewFile(&container{cli.root, inst.ID}), nil
			}
		}
		log.Debugf("Container %v not found in %v", name, cli)
		return nil, plugin.ENOENT
	}
	return nil, plugin.ENOTSUP
}

// List all resources as files.
func (cli *resourcetype) List(ctx context.Context) ([]plugin.Node, error) {
	switch cli.typename {
	case "container":
		containers, err := cli.cachedContainerList(ctx)
		if err != nil {
			return nil, err
		}
		log.Debugf("Listing %v containers in %v", len(containers), cli)
		keys := make([]plugin.Node, len(containers))
		for i, inst := range containers {
			keys[i] = plugin.NewFile(&container{cli.root, inst.ID})
		}
		return keys, nil
	}
	return nil, plugin.ENOTSUP
}

// A unique string describing the resource type.
func (cli *resourcetype) String() string {
	return cli.root.Name() + "/" + cli.Name()
}

// Name returns the name of the resource type.
func (cli *resourcetype) Name() string {
	return cli.typename
}

// Attr returns attributes of the resource type.
func (cli *resourcetype) Attr(ctx context.Context) (*plugin.Attributes, error) {
	// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
	// TODO: make this more constrained to the specific resource.
	latest := cli.updated
	for _, v := range cli.reqs {
		if updated := v.LastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *resourcetype) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

func (cli *root) cachedContainerList(ctx context.Context) ([]types.Container, error) {
	entry, err := cli.Get(cli.Name())
	var containers []types.Container
	if err == nil {
		log.Debugf("Cache hit on %v", cli.Name())
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&containers)
	} else {
		log.Printf("Cache miss on %v", cli.Name())
		containers, err = cli.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(&containers); err != nil {
			return nil, err
		}
		cli.Set(cli.Name(), data.Bytes())
		cli.updated = time.Now()
	}
	return containers, err
}

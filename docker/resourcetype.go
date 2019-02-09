package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type resourcetype struct {
	*client.Client
	plugin.EntryT
	reqs sync.Map
}

func newResourceTypes(cli *root) map[string]*resourcetype {
	resourcetypes := make(map[string]*resourcetype)
	// Use individual caches for slower resources like volumes to control the timeout.
	for _, name := range []string{"container"} {
		resourcetypes[name] = &resourcetype{Client: cli.Client, EntryT: plugin.EntryT{EntryName: name}}
	}
	return resourcetypes
}

// List all instances of the resource type as files.
func (cli *resourcetype) LS(ctx context.Context) ([]plugin.Entry, error) {
	switch cli.Name() {
	case "container":
		containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return nil, err
		}

		log.Debugf("Listing %v containers in %v", len(containers), cli)
		keys := make([]plugin.Entry, len(containers))
		for i, inst := range containers {
			keys[i] = &container{cli.Client, plugin.EntryT{EntryName: inst.ID}, nil}
		}
		return keys, nil
	}
	return nil, plugin.ENOTSUP
}

package docker

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type root struct {
	*client.Client
	plugin.EntryT
	resourcetypes map[string]*resourcetype
}

// Create a new docker client.
func Create(name string, _ interface{}, _ *datastore.MemCache) (plugin.Entry, error) {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cli := &root{dockerCli, plugin.EntryT{EntryName: name}, nil}
	cli.resourcetypes = newResourceTypes(cli)
	return cli, nil
}

func (cli *root) LS(ctx context.Context) ([]plugin.Entry, error) {
	log.Debugf("Listing %v resource types in %v", len(cli.resourcetypes), cli.Name())
	entries := make([]plugin.Entry, 0, len(cli.resourcetypes))
	for _, rt := range cli.resourcetypes {
		entries = append(entries, rt)
	}
	return entries, nil
}

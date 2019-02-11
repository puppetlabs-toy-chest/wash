package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

// Root of the Docker plugin
type Root struct {
	plugin.EntryBase
	client    *client.Client
	resources []plugin.Entry
}

type containers struct {
	plugin.EntryBase
	client *client.Client
}

type container struct {
	plugin.EntryBase
	client *client.Client
}

// Init for root
func (r *Root) Init() error {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	r.EntryBase = plugin.NewEntry("docker")
	r.client = dockerCli

	r.resources = []plugin.Entry{
		&containers{EntryBase: plugin.NewEntry("containers"), client: r.client},
	}

	return nil
}

// LS for root
func (r *Root) LS(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: Have helper for creating EntryTs? E.g. "CreateEntry"
	return r.resources, nil
}

// LS for containers
func (cs *containers) LS(ctx context.Context) ([]plugin.Entry, error) {
	containers, err := cs.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	log.Debugf("Listing %v containers in %v", len(containers), cs)
	keys := make([]plugin.Entry, len(containers))
	for i, inst := range containers {
		keys[i] = &container{EntryBase: plugin.NewEntry(inst.ID), client: cs.client}
	}
	return keys, nil
}

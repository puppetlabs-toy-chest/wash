package docker

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
)

// DOCKER ROOT

// Root of the Docker plugin
type Root struct {
	plugin.EntryBase
	client    *client.Client
	resources []plugin.Entry
}

// Init for root
func (r *Root) Init() error {
	r.EntryBase = plugin.NewEntry("docker")
	r.CacheConfig().TurnOffCaching()

	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	r.client = dockerCli

	r.resources = []plugin.Entry{
		&containers{EntryBase: plugin.NewEntry("containers"), client: r.client},
		&volumes{EntryBase: plugin.NewEntry("volumes"), client: r.client},
	}

	return nil
}

// LS lists the types of resources the Docker plugin exposes.
func (r *Root) LS(ctx context.Context) ([]plugin.Entry, error) {
	return r.resources, nil
}

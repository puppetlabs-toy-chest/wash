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
	resources []plugin.Entry
}

// Init for root
func (r *Root) Init() error {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	r.EntryBase = plugin.NewEntry("docker")
	r.TurnOffCaching()
	r.resources = []plugin.Entry{
		&containers{EntryBase: plugin.NewEntry("containers"), client: dockerCli},
		&volumes{EntryBase: plugin.NewEntry("volumes"), client: dockerCli},
	}

	return nil
}

// List lists the types of resources the Docker plugin exposes.
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	return r.resources, nil
}

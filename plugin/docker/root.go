// Package docker presents a filesystem hierarchy for Docker resources.
//
// It uses local socket access or the DOCKER environment variables to
// access the Docker daemon.
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

	r.EntryBase = plugin.NewEntryBase()
	r.SetName("docker")
	r.DisableDefaultCaching()
	r.resources = []plugin.Entry{
		newContainersDir(dockerCli),
		newVolumesDir(dockerCli),
	}

	return nil
}

// ChildSchemas returns the root's child schema
func (r *Root) ChildSchemas() []plugin.EntrySchema {
	return plugin.ChildSchemas(containersDirBase(), volumesDirBase())
}

// List lists the types of resources the Docker plugin exposes.
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	return r.resources, nil
}

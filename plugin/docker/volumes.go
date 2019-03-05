package docker

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
)

type volumes struct {
	plugin.EntryBase
	client *client.Client
}

// List
func (vs *volumes) List(ctx context.Context) ([]plugin.Entry, error) {
	volumes, err := vs.client.VolumeList(ctx, filters.Args{})
	if err != nil {
		return nil, err
	}

	plugin.Log(ctx, "Listing %v volumes in %v", len(volumes.Volumes), vs)
	keys := make([]plugin.Entry, len(volumes.Volumes))
	for i, inst := range volumes.Volumes {
		if keys[i], err = newVolume(vs.client, inst); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

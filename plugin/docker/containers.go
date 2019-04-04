package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

type containers struct {
	plugin.EntryBase
	client *client.Client
}

// List
func (cs *containers) List(ctx context.Context) ([]plugin.Entry, error) {
	containers, err := cs.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	journal.Record(ctx, "Listing %v containers in %v", len(containers), cs)
	keys := make([]plugin.Entry, len(containers))
	for i, inst := range containers {
		cont := &container{
			EntryBase: cs.NewEntry(inst.ID),
			client:    cs.client,
		}

		startTime := time.Unix(inst.Created, 0)
		attr := plugin.EntryAttributes{}
		attr.
			SetCtime(startTime).
			SetMtime(startTime).
			SetAtime(startTime)
		cont.SetAttributes(attr)

		keys[i] = cont
	}
	return keys, nil
}

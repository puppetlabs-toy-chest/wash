package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/activity"
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

	activity.Record(ctx, "Listing %v containers in %v", len(containers), cs)
	keys := make([]plugin.Entry, len(containers))
	for i, inst := range containers {
		name := inst.ID
		if len(inst.Names) > 0 {
			// The docker API prefixes all names with '/', so remove that.
			// We don't append ID because names must currently be unique in the docker runtime.
			// It's also not clear why 'Names' is an array; `/containers/{id}/json` returns a single
			// Name field while '/containers/json' uses a Names array for each instance. In practice
			// it appears to always be a single name, so take the first as the canonical name.
			name = strings.TrimPrefix(inst.Names[0], "/")
		}
		cont := &container{
			EntryBase: plugin.NewEntry(name),
			id:        inst.ID,
			client:    cs.client,
		}

		startTime := time.Unix(inst.Created, 0)
		attr := plugin.EntryAttributes{}
		attr.
			SetCtime(startTime).
			SetMtime(startTime).
			SetAtime(startTime).
			SetMeta(inst)
		cont.SetAttributes(attr)

		keys[i] = cont
	}
	return keys, nil
}

package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
)

type container struct {
	plugin.EntryBase
	client    *client.Client
	startTime time.Time
}

// Metadata
func (c *container) Metadata(ctx context.Context) (map[string]interface{}, error) {
	// Use raw to also get the container size.
	_, raw, err := c.client.ContainerInspectWithRaw(ctx, c.Name(), true)
	if err != nil {
		return nil, err
	}

	return plugin.ToMetadata(raw), nil
}

// Attr
func (c *container) Attr() plugin.Attributes {
	return plugin.Attributes{
		Ctime: c.startTime,
		Mtime: c.startTime,
		Atime: c.startTime,
	}
}

func (c *container) LS(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{
		&containerMetadata{plugin.NewEntry("metadata.json"), c},
		&containerLogFile{plugin.NewEntry("log"), c.Name(), c.client},
	}, nil
}

package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
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
	dockerCli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	r.EntryBase = plugin.NewEntry("docker")
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

// CONTAINERS DIRECTORY

type containers struct {
	plugin.EntryBase
	client *client.Client
}

// LS
func (cs *containers) LS(ctx context.Context) ([]plugin.Entry, error) {
	containers, err := cs.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	log.Debugf("Listing %v containers in %v", len(containers), cs)
	keys := make([]plugin.Entry, len(containers))
	for i, inst := range containers {
		keys[i] = &container{
			EntryBase: plugin.NewEntry(inst.ID),
			client:    cs.client,
			startTime: time.Unix(inst.Created, 0),
		}
	}
	return keys, nil
}

// CONTAINER DIRECTORY

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

	var metadata map[string]interface{}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
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

// CONTAINER METADATA FILE
type containerMetadata struct {
	plugin.EntryBase
	container *container
}

func (cm *containerMetadata) Open(ctx context.Context) (plugin.SizedReader, error) {
	metadata, err := cm.container.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

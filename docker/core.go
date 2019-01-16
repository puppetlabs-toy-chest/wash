package docker

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"log"
	"time"

	"github.com/allegro/bigcache"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
)

// Client is a docker client.
type Client struct {
	*client.Client
	*bigcache.BigCache
	reqs map[string]RequestRecord
}

// RequestRecord holds arbitrary data and the last time it was updated.
type RequestRecord struct {
	lastUpdate time.Time
	data       interface{}
}

// Create a new docker client.
func Create() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(1 * time.Second))
	if err != nil {
		return nil, err
	}

	reqs := make(map[string]RequestRecord)
	return &Client{cli, cache, reqs}, nil
}

func (cli *Client) cachedContainerList(ctx context.Context) ([]types.Container, error) {
	entry, err := cli.Get("ContainerList")
	var containers []types.Container
	if err == nil {
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&containers)
	} else {
		containers, err = cli.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(&containers); err != nil {
			return nil, err
		}
		cli.Set("ContainerList", data.Bytes())
	}
	return containers, err
}

// Find container by ID.
func (cli *Client) Find(ctx context.Context, name string) (*plugin.Entry, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID == name {
			log.Printf("Found container %v, %v", name, container)
			return &plugin.Entry{Client: cli, Name: container.ID}, nil
		}
	}
	log.Printf("Container %v not found", name)
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *Client) List(ctx context.Context) ([]plugin.Entry, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("Listing %v containers in /docker", len(containers))
	keys := make([]plugin.Entry, len(containers))
	for i, container := range containers {
		keys[i] = plugin.Entry{Client: cli, Name: container.ID}
	}
	return keys, nil
}

// Read gets logs from a container.
func (cli *Client) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	// TODO: store logs in reqs, only query new data. Since we know logs always add,
	// we don't need to worry about invalidating old data.
	opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	return cli.ContainerLogs(ctx, name, opts)
}

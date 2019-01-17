package docker

import (
	"bytes"
	"context"
	"encoding/gob"
	"log"
	"sync"
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
	debug bool
	mux   sync.Mutex
	reqs  map[string]*buffer
}

// Create a new docker client.
func Create(debug bool) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	config := bigcache.DefaultConfig(1 * time.Second)
	config.CleanWindow = 100 * time.Millisecond
	cache, err := bigcache.NewBigCache(config)
	if err != nil {
		return nil, err
	}

	reqs := make(map[string]*buffer)
	return &Client{cli, cache, debug, sync.Mutex{}, reqs}, nil
}

func (cli *Client) log(format string, v ...interface{}) {
	if cli.debug {
		log.Printf(format, v...)
	}
}

func (cli *Client) cachedContainerList(ctx context.Context) ([]types.Container, error) {
	entry, err := cli.Get("ContainerList")
	var containers []types.Container
	if err == nil {
		cli.log("Cache hit in /docker")
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&containers)
	} else {
		cli.log("Cache miss in /docker")
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
			cli.log("Found container %v, %v", name, container)
			return &plugin.Entry{Client: cli, Name: container.ID}, nil
		}
	}
	cli.log("Container %v not found", name)
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *Client) List(ctx context.Context) ([]plugin.Entry, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	cli.log("Listing %v containers in /docker", len(containers))
	keys := make([]plugin.Entry, len(containers))
	for i, container := range containers {
		keys[i] = plugin.Entry{Client: cli, Name: container.ID}
	}
	return keys, nil
}

func (cli *Client) readLog(name string) (*buffer, error) {
	// TODO: investigate log format. Prepending unprintable data, not same format as `docker logs`.
	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	r, err := cli.ContainerLogs(context.Background(), name, opts)
	if err != nil {
		return nil, err
	}
	return newBuffer(name, r), nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, name string) (*plugin.Attributes, error) {
	cli.log("Reading attributes of %v in /docker", name)
	if name == "docker" {
		return &plugin.Attributes{Mtime: time.Now()}, nil
	}

	// TODO: register xattrs.

	// Read the content to figure out how large it is.
	cli.mux.Lock()
	defer cli.mux.Unlock()
	buf, ok := cli.reqs[name]
	if !ok {
		var err error
		buf, err = cli.readLog(name)
		if err != nil {
			return nil, err
		}

		cli.reqs[name] = buf
	}

	lastUpdate := buf.lastUpdate()
	size := uint64(buf.len())
	return &plugin.Attributes{Mtime: lastUpdate, Size: size}, nil
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, name string) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()
	buf, ok := cli.reqs[name]
	if !ok {
		var err error
		buf, err = cli.readLog(name)
		if err != nil {
			return nil, err
		}

		cli.reqs[name] = buf
	}

	return buf, nil
}

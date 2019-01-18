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
	debug   bool
	mux     sync.Mutex
	reqs    map[string]*buffer
	updated time.Time
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
	return &Client{cli, cache, debug, sync.Mutex{}, reqs, time.Now()}, nil
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
		cli.updated = time.Now()
	}
	return containers, err
}

func (cli *Client) cachedContainerInspect(ctx context.Context, name string) (*types.ContainerJSON, error) {
	entry, err := cli.Get(name)
	var container types.ContainerJSON
	if err == nil {
		cli.log("Cache hit in /docker/%v", name)
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&container)
	} else {
		cli.log("Cache miss in /docker/%v", name)
		container, err = cli.ContainerInspect(ctx, name)
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(&container); err != nil {
			return nil, err
		}
		cli.Set(name, data.Bytes())
	}
	return &container, err
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

func (cli *Client) readLog(ctx context.Context, name string) (*buffer, error) {
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

	c, err := cli.cachedContainerInspect(ctx, name)
	if err != nil {
		return nil, err
	}
	return newBuffer(name, r, c.Config.Tty), nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, name string) (*plugin.Attributes, error) {
	cli.log("Reading attributes of %v in /docker", name)
	if name == "docker" {
		// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
		log.Printf("Getting attr of /docker")
		latest := cli.updated
		for _, v := range cli.reqs {
			if updated := v.lastUpdate(); updated.After(latest) {
				latest = updated
			}
		}
		log.Printf("Mtime: %v", latest)
		return &plugin.Attributes{Mtime: latest}, nil
	}

	// TODO: register xattrs.

	// Read the content to figure out how large it is.
	cli.mux.Lock()
	defer cli.mux.Unlock()
	buf, ok := cli.reqs[name]
	if !ok {
		var err error
		buf, err = cli.readLog(ctx, name)
		if err != nil {
			return nil, err
		}

		cli.reqs[name] = buf
	}

	return &plugin.Attributes{Mtime: buf.lastUpdate(), Size: uint64(buf.len())}, nil
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, name string) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()
	buf, ok := cli.reqs[name]
	if !ok {
		var err error
		buf, err = cli.readLog(ctx, name)
		if err != nil {
			return nil, err
		}

		cli.reqs[name] = buf
	}
	go func() {
		buf.stream()
	}()
	// Wait for some output to buffer
	time.Sleep(500 * time.Millisecond)

	return buf, nil
}

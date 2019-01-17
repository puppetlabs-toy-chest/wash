package docker

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"io/ioutil"
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
	reqs  map[string]*RequestRecord
	debug bool
}

// RequestRecord holds arbitrary data and the last time it was updated.
type RequestRecord struct {
	lastUpdate time.Time
	data       []byte
	reader     *bytes.Reader
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

	reqs := make(map[string]*RequestRecord)
	return &Client{cli, cache, reqs, debug}, nil
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

func (cli *Client) readLog(ctx context.Context, name string) ([]byte, error) {
	opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	r, err := cli.ContainerLogs(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, name string) (*plugin.Attributes, error) {
	cli.log("Reading attributes of %v in /docker", name)
	if name == "docker" {
		// Return attributes for the client, i.e. when was the last update. TODO: make it so
		return &plugin.Attributes{Mtime: time.Now()}, nil
	}

	// Read the content to figure out how large it is.
	buf, err := cli.readLog(ctx, name)
	if err != nil {
		return nil, err
	}
	size := uint64(len(buf))

	req, ok := cli.reqs[name]
	if !ok {
		req = &RequestRecord{}
		cli.reqs[name] = req
	}
	req.data = buf
	if req.reader != nil {
		// Reset the buffer so any open FileHandles will get the updated data.
		req.reader.Reset(buf)
	}

	return &plugin.Attributes{Mtime: time.Now(), Size: size}, nil
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, name string) (io.ReaderAt, error) {
	// TODO: store logs in reqs, only query new data. Since we know logs always add,
	// we don't need to worry about invalidating old data.
	// TODO: need an additional callback for attributes that updates from ContainerLogOptions
	// and reports whether there were any changes.
	req, ok := cli.reqs[name]
	if !ok {
		buf, err := cli.readLog(ctx, name)
		if err != nil {
			return nil, err
		}
		req = &RequestRecord{data: buf}
		cli.reqs[name] = req
	}

	req.reader = bytes.NewReader(req.data)
	return req.reader, nil
}

package docker

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
)

// Client is a docker client.
type Client struct {
	*client.Client
	*bigcache.BigCache
	debug   bool
	mux     sync.Mutex
	reqs    map[string]*datastore.StreamBuffer
	updated time.Time
	root    string
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

	reqs := make(map[string]*datastore.StreamBuffer)
	return &Client{cli, cache, debug, sync.Mutex{}, reqs, time.Now(), "docker"}, nil
}

func (cli *Client) log(format string, v ...interface{}) {
	if cli.debug {
		log.Printf(format, v...)
	}
}

// Find container by ID.
func (cli *Client) Find(ctx context.Context, parent *plugin.Dir, name string) (plugin.Node, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID == name {
			cli.log("Found container %v, %v", name, container)
			return &plugin.File{Client: cli, Parent: parent, Name: container.ID}, nil
		}
	}
	cli.log("Container %v not found", name)
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *Client) List(ctx context.Context, parent *plugin.Dir) ([]plugin.Node, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	cli.log("Listing %v containers in /docker", len(containers))
	keys := make([]plugin.Node, len(containers))
	for i, container := range containers {
		keys[i] = &plugin.File{Client: cli, Parent: parent, Name: container.ID}
	}
	return keys, nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, name string) (*plugin.Attributes, error) {
	cli.log("Reading attributes of %v in /docker", name)
	if name == cli.root {
		// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
		latest := cli.updated
		for _, v := range cli.reqs {
			if updated := v.LastUpdate(); updated.After(latest) {
				latest = updated
			}
		}
		log.Printf("Mtime: %v", latest)
		return &plugin.Attributes{Mtime: latest, Valid: 100 * time.Millisecond}, nil
	}

	// Read the content to figure out how large it is.
	cli.mux.Lock()
	defer cli.mux.Unlock()
	if buf, ok := cli.reqs[name]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: 100 * time.Millisecond}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: 1 * time.Second}, nil
}

// Xattr returns a map of extended attributes.
func (cli *Client) Xattr(ctx context.Context, name string) (map[string][]byte, error) {
	raw, err := cli.cachedContainerInspectRaw(ctx, name)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	d := make(map[string][]byte)
	for k, v := range data {
		d[k], err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (cli *Client) readLog(name string) (io.ReadCloser, error) {
	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	return cli.ContainerLogs(context.Background(), name, opts)
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, name string) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()

	c, err := cli.cachedContainerInspect(ctx, name)
	if err != nil {
		return nil, err
	}

	buf, ok := cli.reqs[name]
	if !ok {
		buf = datastore.NewBuffer(name)
		cli.reqs[name] = buf
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(cli.readLog, buffered, c.Config.Tty)
	}()
	// Wait for some output to buffer.
	<-buffered
	// Wait a short time for reading the stream.
	time.Sleep(100 * time.Millisecond)

	return buf, nil
}

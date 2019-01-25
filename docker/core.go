package docker

import (
	"context"
	"encoding/binary"
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

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const (
	validDuration = 100 * time.Millisecond
	headerLen     = 8
	headerSizeIdx = 4
)

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
func (cli *Client) Find(ctx context.Context, parent *plugin.Dir, name string) (plugin.Entry, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID == name {
			cli.log("Found container %v, %v", name, container)
			return plugin.NewFile(cli, parent, container.ID), nil
		}
	}
	cli.log("Container %v not found", name)
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *Client) List(ctx context.Context, parent *plugin.Dir) ([]plugin.Entry, error) {
	containers, err := cli.cachedContainerList(ctx)
	if err != nil {
		return nil, err
	}
	cli.log("Listing %v containers in /docker", len(containers))
	keys := make([]plugin.Entry, len(containers))
	for i, container := range containers {
		keys[i] = plugin.NewFile(cli, parent, container.ID)
	}
	return keys, nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, node plugin.Entry) (*plugin.Attributes, error) {
	if node == nil || node.Name() == cli.root {
		// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
		latest := cli.updated
		for _, v := range cli.reqs {
			if updated := v.LastUpdate(); updated.After(latest) {
				latest = updated
			}
		}
		return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
	}

	cli.log("Reading attributes of %v in /docker", node.Name())
	// Read the content to figure out how large it is.
	cli.mux.Lock()
	defer cli.mux.Unlock()
	if buf, ok := cli.reqs[node.Name()]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *Client) Xattr(ctx context.Context, node plugin.Entry) (map[string][]byte, error) {
	raw, err := cli.cachedContainerInspectRaw(ctx, node.Name())
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

// Removes multiplex headers. Returns the new buffer length after compressing input,
// and the new writeIndex that also includes unprocessed data.
func processMultiplexedStreams(data []byte, writeIndex int) (int, int) {
	// Do extra processing to strip out multiplex prefix. Format is of the form
	//   [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}[]byte{OUTPUT}
	// readIndex represents how far we've processed the buffered input.
	// writeIndex is the end of the buffered input.
	// newLen represents the end of processed input, which will trail readIndex as we append new processed input.
	newLen, readIndex, capacity := len(data), len(data), cap(data)
	for writeIndex-readIndex >= headerLen {
		// Get the remaining unprocessed buffer.
		buf := data[readIndex:writeIndex]

		// Read the next frame.
		frameSize := int(binary.BigEndian.Uint32(buf[headerSizeIdx : headerSizeIdx+4]))

		// Stop if the frame is larger than the remaining unprocessed buffer.
		if headerLen+frameSize > len(buf) {
			break
		}

		// Append frame to processed input and increment newLen.
		// This space can later be used for coloring output based on stream.
		copy(data[newLen:capacity], buf[headerLen:headerLen+frameSize])
		readIndex += headerLen + frameSize
		newLen += frameSize
	}

	// Append any remaining input to the processed input.
	buf := data[readIndex:writeIndex]
	copy(data[newLen:capacity], buf)
	writeIndex = newLen + len(buf)
	return newLen, writeIndex
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
func (cli *Client) Open(ctx context.Context, node plugin.Entry) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()

	c, err := cli.cachedContainerInspect(ctx, node.Name())
	if err != nil {
		return nil, err
	}

	buf, ok := cli.reqs[node.Name()]
	if !ok {
		// Only do additional processing if container is not running with tty.
		postProcessor := processMultiplexedStreams
		if c.Config.Tty {
			postProcessor = nil
		}
		buf = datastore.NewBuffer(node.Name(), postProcessor)
		cli.reqs[node.Name()] = buf
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(cli.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return buf, nil
}

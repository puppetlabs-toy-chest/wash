package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type container struct {
	*resourcetype
	name string
}

const (
	headerLen     = 8
	headerSizeIdx = 4
)

// String returns a unique representation of the project.
func (inst *container) String() string {
	return inst.root.Name() + "/container/" + inst.Name()
}

// Name returns the container's ID.
func (inst *container) Name() string {
	return inst.name
}

// Attr returns attributes of the named resource.
func (inst *container) Attr(ctx context.Context) (*plugin.Attributes, error) {
	log.Debugf("Reading attributes of %v", inst)
	// Read the content to figure out how large it is.
	if v, ok := inst.reqs.Load(inst.name); ok {
		buf := v.(*datastore.StreamBuffer)
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size())}, nil
	}

	// Prefetch content for next time.
	go plugin.PrefetchOpen(inst)

	return &plugin.Attributes{Mtime: inst.updated}, nil
}

// Xattr returns a map of extended attributes.
func (inst *container) Xattr(ctx context.Context) (map[string][]byte, error) {
	raw, err := inst.cache.CachedJSON(inst.String(), func() ([]byte, error) {
		_, raw, err := inst.ContainerInspectWithRaw(ctx, inst.name, true)
		return raw, err
	})
	if err != nil {
		return nil, err
	}

	return plugin.JSONToJSONMap(raw)
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

func (inst *container) readLog() (io.ReadCloser, error) {
	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	return inst.ContainerLogs(context.Background(), inst.name, opts)
}

// Open gets logs from a container.
func (inst *container) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	c, err := inst.cachedContainerInspect(ctx)
	if err != nil {
		return nil, err
	}

	// Only do additional processing if container is not running with tty.
	postProcessor := processMultiplexedStreams
	if c.Config.Tty {
		postProcessor = nil
	}
	buf := datastore.NewBuffer(inst.name, postProcessor)

	if v, ok := inst.reqs.LoadOrStore(inst.name, buf); ok {
		buf = v.(*datastore.StreamBuffer)
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(inst.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return buf, nil
}

func (inst *container) cachedContainerInspect(ctx context.Context) (*types.ContainerJSON, error) {
	entry, err := inst.cache.Get(inst.String())
	var container types.ContainerJSON
	if err == nil {
		log.Debugf("Cache hit on %v", inst)
		rdr := bytes.NewReader(entry)
		err = json.NewDecoder(rdr).Decode(&container)
	} else {
		log.Printf("Cache miss on %v", inst)
		var raw []byte
		container, raw, err = inst.ContainerInspectWithRaw(ctx, inst.name, true)
		if err != nil {
			return nil, err
		}

		inst.cache.Set(inst.name, raw)
	}

	return &container, err
}

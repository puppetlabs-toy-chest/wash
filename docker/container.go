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
	*root
	name string
}

// Name returns the container's ID.
func (inst *container) Name() string {
	return inst.name
}

// Attr returns attributes of the named resource.
func (inst *container) Attr(ctx context.Context) (*plugin.Attributes, error) {
	log.Debugf("Reading attributes of %v in /docker", inst.name)
	// Read the content to figure out how large it is.
	inst.mux.Lock()
	defer inst.mux.Unlock()
	if buf, ok := inst.reqs[inst.name]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: inst.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (inst *container) Xattr(ctx context.Context) (map[string][]byte, error) {
	raw, err := inst.cachedContainerInspectRaw(ctx, inst.name)
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

func (inst *container) readLog(name string) (io.ReadCloser, error) {
	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	return inst.ContainerLogs(context.Background(), name, opts)
}

// Open gets logs from a container.
func (inst *container) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	inst.mux.Lock()
	defer inst.mux.Unlock()

	c, err := inst.cachedContainerInspect(ctx, inst.name)
	if err != nil {
		return nil, err
	}

	buf, ok := inst.reqs[inst.name]
	if !ok {
		// Only do additional processing if container is not running with tty.
		postProcessor := processMultiplexedStreams
		if c.Config.Tty {
			postProcessor = nil
		}
		buf = datastore.NewBuffer(inst.name, postProcessor)
		inst.reqs[inst.name] = buf
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(inst.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return buf, nil
}

func (inst *container) cachedContainerInspect(ctx context.Context, name string) (*types.ContainerJSON, error) {
	entry, err := inst.Get(name)
	var container types.ContainerJSON
	if err == nil {
		log.Debugf("Cache hit in /docker/%v", name)
		rdr := bytes.NewReader(entry)
		err = json.NewDecoder(rdr).Decode(&container)
	} else {
		log.Debugf("Cache miss in /docker/%v", name)
		var raw []byte
		container, raw, err = inst.ContainerInspectWithRaw(ctx, name, true)
		if err != nil {
			return nil, err
		}

		inst.Set(name, raw)
	}

	return &container, err
}

func (inst *container) cachedContainerInspectRaw(ctx context.Context, name string) ([]byte, error) {
	entry, err := inst.Get(name)
	if err == nil {
		log.Debugf("Cache hit in /docker/%v", name)
		return entry, nil
	}

	log.Debugf("Cache miss in /docker/%v", name)
	_, raw, err := inst.ContainerInspectWithRaw(ctx, name, true)
	if err != nil {
		return nil, err
	}

	inst.Set(name, raw)
	return raw, nil
}

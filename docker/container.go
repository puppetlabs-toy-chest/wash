package docker

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
)

type container struct {
	*client.Client
	plugin.EntryT
	buf *datastore.StreamBuffer
}

const (
	headerLen     = 8
	headerSizeIdx = 4
)

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
	}
	return inst.ContainerLogs(context.Background(), inst.Name(), opts)
}

func (inst *container) Metadata(ctx context.Context) (interface{}, error) {
	// Use raw to also get the container size.
	container, _, err := inst.ContainerInspectWithRaw(ctx, inst.Name(), true)
	return container, err
}

func (inst *container) Size() uint64 {
	if inst.buf != nil {
		return uint64(inst.buf.Size())
	}
	return 0
}

// Open gets logs from a container.
func (inst *container) Open(ctx context.Context) (io.ReaderAt, error) {
	tty := false
	// TODO: how should we get the cached version from the engine?
	if meta, err := inst.Metadata(ctx); err == nil {
		if c, ok := meta.(*types.ContainerJSON); ok {
			tty = c.Config.Tty
		}
	} else {
		return nil, err
	}

	// Only do additional processing if container is not running with tty.
	postProcessor := processMultiplexedStreams
	if tty {
		postProcessor = nil
	}

	// TODO: switch to non-streaming output
	inst.buf = datastore.NewBuffer(inst.Name(), postProcessor)

	buffered := make(chan bool)
	go func() {
		inst.buf.Stream(inst.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return inst.buf, nil
}

// Package datastore provides primitives useful for locally caching data related to remote resources.
package datastore

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"sync"
	"time"
)

// StreamBuffer implements a streaming buffer that can be closed and re-opened.
// Includes locking on all operations so that they can safely be performed while data
// is being streamed to its internal buffer. Implements interfaces io.ReaderAt and io.Closer.
type StreamBuffer struct {
	mux       sync.Mutex
	name      string
	data      []byte
	input     io.ReadCloser
	reader    *bytes.Reader
	update    time.Time
	size      int
	streaming int
}

const (
	minRead       = 4096
	slowLimit     = 64 * 1024 * 1024
	headerLen     = 8
	headerSizeIdx = 4
)

// NewBuffer instantiates a new streaming buffer for the named resource.
func NewBuffer(name string) *StreamBuffer {
	b := StreamBuffer{name: name, data: make([]byte, 0, minRead), update: time.Now()}
	b.reader = bytes.NewReader(b.data)
	return &b
}

func (b *StreamBuffer) incr() int {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.streaming++
	return b.streaming
}

func (b *StreamBuffer) decr() int {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.streaming--
	return b.streaming
}

// Stream reads from a reader instantiated with the specified callback. Stores all data in an
// internal buffer. Sends a confirmation on the provided channel when some data has been buffered.
// Whenever new data is injested, locks and updates the buffer's reader with a new slice.
func (b *StreamBuffer) Stream(cb func(string) (io.ReadCloser, error), confirm chan bool, tty bool) {
	if count := b.incr(); count > 1 {
		// Only initiate streaming if this is the first request.
		confirm <- true
		close(confirm)
		return
	}

	var err error
	b.input, err = cb(b.name)
	confirm <- true
	close(confirm)
	if err != nil {
		log.Printf("Buffer setup failed: %v", err)
		b.decr()
		return
	}

	// Track writeIndex. For TTY containers, this will always match len(b.data).
	// For non-TTY, it will be larger than len(b.data) when a partial frame is buffered.
	writeIndex := 0
	for {
		// Grow the buffer as needed. Start out quadrupling, but slow down when storing tens of megabytes.
		if spare := cap(b.data) - writeIndex; spare < minRead {
			growBy := 3 * cap(b.data)
			if growBy > slowLimit {
				growBy = slowLimit
			}
			ndata := make([]byte, writeIndex, cap(b.data)+growBy)
			copy(ndata, b.data)
			b.data = ndata

			// Update the buffer so we can release the old array.
			b.mux.Lock()
			b.reader.Reset(b.data)
			b.mux.Unlock()
		}

		// Read data. This may block while waiting for new input.
		capacity := cap(b.data)
		log.Printf("Reading %v [%v/%v]", b.name, writeIndex, capacity)
		bytesRead, err := b.input.Read(b.data[writeIndex:capacity])
		if bytesRead < 0 {
			panic("buffer: readFrom returned negative count from Read")
		}
		writeIndex += bytesRead
		log.Printf("Read %v [%v/%v]", b.name, writeIndex, capacity)

		newLen := writeIndex
		if !tty {
			// Do extra processing to strip out multiplex prefix. Format is of the form
			//   [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}[]byte{OUTPUT}
			// readIndex represents how far we've processed the buffered input.
			// writeIndex is the end of the buffered input.
			// newLen represents the end of processed input, which will trail readIndex as we append new processed input.
			readIndex := len(b.data)
			for newLen = len(b.data); writeIndex-readIndex >= headerLen; {
				// Get the remaining unprocessed buffer.
				buf := b.data[readIndex:writeIndex]

				// Read the next frame.
				frameSize := int(binary.BigEndian.Uint32(buf[headerSizeIdx : headerSizeIdx+4]))

				// Stop if the frame is larger than the remaining unprocessed buffer.
				if headerLen+frameSize > len(buf) {
					break
				}

				// Append frame to processed input and increment newLen.
				// This space can later be used for coloring output based on stream.
				copy(b.data[newLen:capacity], buf[headerLen:headerLen+frameSize])
				readIndex += headerLen + frameSize
				newLen += frameSize
			}

			// Append any remaining input to the processed input.
			buf := b.data[readIndex:writeIndex]
			copy(b.data[newLen:capacity], buf)
			writeIndex = newLen + len(buf)
		}

		// Update reader with new slice.
		b.mux.Lock()
		b.data = b.data[:newLen]
		b.reader.Reset(b.data)
		b.size = newLen
		b.update = time.Now()
		b.mux.Unlock()

		if err == io.EOF {
			b.input.Close()
			break
		} else if err != nil {
			log.Printf("Read failed, perhaps connection or file was closed: %v", err)
			// If the connection was closed explicitly, clear data.
			b.mux.Lock()
			b.data = b.data[:0]
			b.reader.Reset(b.data)
			// Don't reset size on close.
			b.mux.Unlock()
			break
		}
	}
}

// ReadAt implements the ReaderAt interface. Prevents buffer updates during read.
func (b *StreamBuffer) ReadAt(p []byte, off int64) (int, error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.reader.ReadAt(p, off)
}

// Close implements the Closer interface. Includes reference counting of
// times a stream was requested and only closes the input when that reaches 0.
func (b *StreamBuffer) Close() error {
	if count := b.decr(); count == 0 {
		return b.input.Close()
	}
	return nil
}

// Size returns the size of buffered data. If the stream has been closed, reports
// the last known size.
func (b *StreamBuffer) Size() int {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.size
}

// LastUpdate reports the last time there was an update from the stream.
func (b *StreamBuffer) LastUpdate() time.Time {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.update
}

package docker

import (
	"bytes"
	"io"
	"log"
	"sync"
	"time"
)

// Implements a streaming buffer. Implements io.ReaderAt.
type buffer struct {
	mux       sync.Mutex
	name      string
	data      []byte
	input     io.ReadCloser
	reader    *bytes.Reader
	update    time.Time
	streaming int
	tty       bool
}

const minRead = 512
const slowLimit = 64 * 1024 * 1024

func newBuffer(name string, r io.ReadCloser, tty bool) *buffer {
	b := buffer{name: name, data: make([]byte, 0, minRead), input: r, update: time.Now(), tty: tty}
	b.reader = bytes.NewReader(b.data)
	return &b
}

func (b *buffer) incr() int {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.streaming++
	return b.streaming
}

func (b *buffer) decr() int {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.streaming--
	return b.streaming
}

// Reads from the specified reader. Stores all data in an internal buffer.
// Whenever new data is injested, locks and updates the buffer's reader with a new slice.
func (b *buffer) stream() {
	if count := b.incr(); count > 1 {
		// Only initiate streaming if this is the first request.
		return
	}

	for {
		// TODO: reimplement stdcopy with control over the buffer
		// Grow the buffer as needed. Start out quadrupling, but slow down when storing tens of megabytes.
		if spare := cap(b.data) - len(b.data); spare < minRead {
			growBy := 3 * cap(b.data)
			if growBy > slowLimit {
				growBy = slowLimit
			}
			ndata := make([]byte, len(b.data), cap(b.data)+growBy)
			copy(ndata, b.data)
			b.data = ndata

			// Update the buffer so we can release the old array.
			b.mux.Lock()
			b.reader.Reset(b.data)
			b.mux.Unlock()
		}

		// Read data. This may block while waiting for new input.
		i, c := len(b.data), cap(b.data)
		log.Printf("Reading %v [%v/%v]", b.name, i, c)
		m, err := b.input.Read(b.data[i:c])
		if m < 0 {
			panic("buffer: readFrom returned negative count from Read")
		}
		log.Printf("Read %v [%v/%v]", b.name, i+m, c)

		// Update reader with new slice.
		b.mux.Lock()
		b.data = b.data[:i+m]
		b.reader.Reset(b.data)
		b.update = time.Now()
		b.mux.Unlock()

		if err == io.EOF {
			b.input.Close()
			break
		} else if err != nil {
			log.Printf("Read failed, perhaps connection or file was closed")
			// If the connection was closed explicitly, clear data.
			b.mux.Lock()
			b.data = b.data[:0]
			b.reader.Reset(b.data)
			b.mux.Unlock()
			break
		}
	}
}

// ReadAt implements the ReaderAt interface. Prevents buffer updates during read.
func (b *buffer) ReadAt(p []byte, off int64) (int, error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.reader.ReadAt(p, off)
}

func (b *buffer) Close() {
	if count := b.decr(); count == 0 {
		b.input.Close()
	}
}

func (b *buffer) len() int64 {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.reader.Size()
}

func (b *buffer) lastUpdate() time.Time {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.update
}

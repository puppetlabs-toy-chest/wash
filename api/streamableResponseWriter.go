package api

import (
	"io"
	"net/http"
)

// Inspired by Docker's WriteFlusher: https://github.com/moby/moby/blob/17.05.x/pkg/ioutils/writeflusher.go
type flushableWriter interface {
	io.Writer
	http.Flusher
}
type streamableResponseWriter struct {
	flushableWriter
}

// Write flushes the data immediately after every write operation
func (wf *streamableResponseWriter) Write(b []byte) (n int, err error) {
	n, err = wf.flushableWriter.Write(b)
	wf.Flush()
	return n, err
}

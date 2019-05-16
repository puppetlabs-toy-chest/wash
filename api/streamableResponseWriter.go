package api

import (
	"context"
	"io"
	"net/http"

	"github.com/puppetlabs/wash/activity"
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

type cleanupFunc = func() error

func streamCleanup(ctx context.Context, desc string, cleanup cleanupFunc) {
	go func() {
		<-ctx.Done()
		activity.Record(ctx, "API: %v closed by completed context: %v", desc, cleanup())
	}()
}

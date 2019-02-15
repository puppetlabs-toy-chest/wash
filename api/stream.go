package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

func streamHandler(w http.ResponseWriter, r *http.Request) {
	path := mux.Vars(r)["path"]
	log.Infof("API: Stream %v", path)

	entry, err := getEntryFromPath(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	pipe, ok := entry.(plugin.Pipe)
	if !ok {
		http.Error(w, fmt.Sprintf("Entry %v does not support the stream command", path), http.StatusNotFound)
		return
	}

	rdr, err := pipe.Stream(r.Context())

	// TODO: Definitely figure out the error handling at some point
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not stream %v: %v\n", path, err), http.StatusInternalServerError)
		return
	}

	f, ok := w.(flushableWriter)
	if !ok {
		http.Error(w, fmt.Sprintf("Could not stream %v, response handler does not support flushing", path), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	// Ensure every write is a flush, and do an initial flush to send the header.
	wf := &streamableResponseWriter{f}
	f.Flush()

	if closer, ok := rdr.(io.Closer); ok {
		// If a ReadCloser, ensure it's closed when the context is cancelled.
		go func() {
			<-r.Context().Done()
			closer.Close()
		}()
	}
	if _, err := io.Copy(wf, rdr); err != nil {
		// Common for copy to error when the caller closes the connection.
		log.Debugf("Errored streaming response for entry %v: %v", path, err)
	}
}

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

package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

var streamHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodGet {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodGet})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Stream %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	pipe, ok := entry.(plugin.Pipe)
	if !ok {
		return unsupportedActionResponse(path, streamAction)
	}

	f, ok := w.(flushableWriter)
	if !ok {
		return unknownErrorResponse(fmt.Errorf("Cannot stream %v, response handler does not support flushing", path))
	}

	rdr, err := pipe.Stream(r.Context())

	if err != nil {
		return erroredActionResponse(path, streamAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	// Ensure every write is a flush, and do an initial flush to send the header.
	wf := &streamableResponseWriter{f}
	f.Flush()

	if closer, ok := rdr.(io.Closer); ok {
		// If a ReadCloser, ensure it's closed when the context is cancelled.
		go func() {
			<-r.Context().Done()
			plugin.LogErr(closer.Close(), "Error closing Stream() stream")
		}()
	}
	if _, err := io.Copy(wf, rdr); err != nil {
		// Common for copy to error when the caller closes the connection.
		log.Debugf("Errored streaming response for entry %v: %v", path, err)
	}

	return nil
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

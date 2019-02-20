package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

type execBody struct {
	Cmd  string             `json:"cmd"`
	Args []string           `json:"args"`
	Opts plugin.ExecOptions `json:"opts"`
}

var execHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodPost {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodPost})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Exec %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	exec, ok := entry.(plugin.Execable)
	if !ok {
		return unsupportedActionResponse(path, execAction)
	}

	if r.Body == nil {
		return badRequestResponse(r.URL.Path, "Please send a JSON request body")
	}

	var body execBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return badRequestResponse(r.URL.Path, err.Error())
	}

	// TODO: This and the stream endpoint have some shared code for streaming
	// responses. That should be moved to a separate helper at some point.
	f, ok := w.(flushableWriter)
	if !ok {
		return unknownErrorResponse(fmt.Errorf("Cannot stream %v, response handler does not support flushing", path))
	}

	rdr, err := exec.Exec(r.Context(), body.Cmd, body.Args, body.Opts)
	if err != nil {
		return erroredActionResponse(path, execAction, err.Error())
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

	return nil
}

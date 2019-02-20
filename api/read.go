package api

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var readHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	path := mux.Vars(r)["path"]
	log.Infof("API: Read %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	readable, ok := entry.(plugin.Readable)
	if !ok {
		return unsupportedActionResponse(path, readAction)
	}

	content, err := plugin.CachedOpen(readable, toID(path), r.Context())

	if err != nil {
		return erroredActionResponse(path, readAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	if rdr, ok := content.(io.Reader); ok {
		io.Copy(w, rdr)
	} else {
		io.Copy(w, io.NewSectionReader(content, 0, content.Size()))
	}

	return nil
}

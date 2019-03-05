package api

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var readHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodGet {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodGet})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Read %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	if !plugin.ReadAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ReadAction)
	}

	content, err := plugin.CachedOpen(r.Context(), entry.(plugin.Readable), toID(path))

	if err != nil {
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	n, err := io.Copy(w, io.NewSectionReader(content, 0, content.Size()))
	if n != content.Size() {
		log.Warnf("Read incomplete %v/%v", n, content.Size())
	}
	if err != nil {
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}

	return nil
}

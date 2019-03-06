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

	ctx := r.Context()
	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.ReadAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ReadAction)
	}

	plugin.Log(ctx, "API: Read %v", path)
	content, err := plugin.CachedOpen(ctx, entry.(plugin.Readable), toID(path))

	if err != nil {
		plugin.Log(ctx, "API: Read %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}
	plugin.Log(ctx, "API: Reading %v", path)

	w.WriteHeader(http.StatusOK)
	n, err := io.Copy(w, io.NewSectionReader(content, 0, content.Size()))
	if n != content.Size() {
		log.Warnf("Read incomplete %v/%v", n, content.Size())
		plugin.Log(ctx, "API: Reading %v incomplete: %v/%v", path, n, content.Size())
	}
	if err != nil {
		plugin.Log(ctx, "API: Reading %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}

	plugin.Log(ctx, "API: Reading %v complete", path)
	return nil
}

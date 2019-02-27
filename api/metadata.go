package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var metadataHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodGet {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodGet})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Metadata %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	if !plugin.MetadataAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.MetadataAction)
	}

	metadata, err := plugin.CachedMetadata(r.Context(), entry.(plugin.Resource), toID(path))

	if err != nil {
		return erroredActionResponse(path, plugin.MetadataAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}

	return nil
}

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

	ctx := r.Context()
	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.MetadataAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.MetadataAction)
	}

	plugin.Log(ctx, "API: Metadata %v", path)
	metadata, err := plugin.CachedMetadata(ctx, entry.(plugin.Resource), toID(path))

	if err != nil {
		plugin.Log(ctx, "API: Metadata %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.MetadataAction, err.Error())
	}
	plugin.Log(ctx, "API: Metadata %v %+v", path, metadata)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		plugin.Log(ctx, "API: Metadata marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}

	plugin.Log(ctx, "API: Metadata %v complete", path)
	return nil
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var metadataHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodGet {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodGet})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Metadata %v", path)

	jnl := journal.NamedJournal{ID: r.FormValue(apitypes.JournalID)}
	ctx := context.WithValue(r.Context(), plugin.Journal, jnl)

	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.MetadataAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.MetadataAction)
	}

	jnl.Log("API: Metadata %v", path)
	metadata, err := plugin.CachedMetadata(ctx, entry.(plugin.Resource), toID(path))

	if err != nil {
		jnl.Log("API: Metadata %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.MetadataAction, err.Error())
	}
	jnl.Log("API: Metadata %v %+v", path, metadata)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		jnl.Log("API: Metadata marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}

	jnl.Log("API: Metadata %v complete", path)
	return nil
}

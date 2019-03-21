package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

var metadataHandler handler = func(w http.ResponseWriter, r *http.Request, path string) *errorResponse {
	ctx := r.Context()
	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.MetadataAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.MetadataAction)
	}

	journal.Record(ctx, "API: Metadata %v", path)
	metadata, err := plugin.CachedMetadata(ctx, entry.(plugin.Resource))

	if err != nil {
		journal.Record(ctx, "API: Metadata %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.MetadataAction, err.Error())
	}
	journal.Record(ctx, "API: Metadata %v %+v", path, metadata)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		journal.Record(ctx, "API: Metadata marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}

	journal.Record(ctx, "API: Metadata %v complete", path)
	return nil
}

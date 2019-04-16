package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:response
//nolint:deadcode,unused
type entryMetadata struct {
	EntryMetadata plugin.EntryMetadata
}

// swagger:route GET /fs/metadata metadata getMetadata
//
// Get metadata
//
// Get metadata about the specified entry.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: entryMetadata
//       404: errorResp
//       500: errorResp
var metadataHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(ctx, r)
	if errResp != nil {
		return errResp
	}

	activity.Record(ctx, "API: Metadata %v", path)
	metadata, err := plugin.CachedMetadata(ctx, entry)

	if err != nil {
		activity.Record(ctx, "API: Metadata %v errored: %v", path, err)
		return unknownErrorResponse(err)
	}
	activity.Record(ctx, "API: Metadata %v %+v", path, metadata)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		activity.Record(ctx, "API: Metadata marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}

	activity.Record(ctx, "API: Metadata %v complete", path)
	return nil
}

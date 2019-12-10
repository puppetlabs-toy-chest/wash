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
	JSONObject plugin.JSONObject
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
var metadataHandler = handler{fn: func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}

	metadata, err := plugin.Metadata(ctx, entry)

	if err != nil {
		return unknownErrorResponse(err)
	}
	activity.Record(ctx, "API: Metadata %v %+v", path, metadata)

	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}
	return nil
}}

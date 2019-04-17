package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/activity"
)

// swagger:route GET /fs/info info entryInfo
//
// Info about entry at path
//
// Returns an Entry object describing the given path.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: Entry
//       400: errorResp
//       404: errorResp
//       500: errorResp
var infoHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(ctx, r)
	if errResp != nil {
		return errResp
	}

	activity.Record(ctx, "API: Info %v", path)
	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	// TODO: Include the entry's full metadata?
	apiEntry := toAPIEntry(entry)
	apiEntry.Path = path
	if err := jsonEncoder.Encode(&apiEntry); err != nil {
		activity.Record(ctx, "API: Info: marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal %v: %v", path, err))
	}

	activity.Record(ctx, "API: Info %v complete", path)
	return nil
}

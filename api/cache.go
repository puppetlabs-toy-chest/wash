package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:route DELETE /cache cache cacheDelete
//
// Remove items from the cache
//
// Removes the specified entry and its children from the cache.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200:
//       500: errorResp
var cacheHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	path, errResp := getWashPathFromRequest(r)
	if errResp != nil {
		return errResp
	}

	deleted, err := plugin.ClearCacheFor(path)
	if err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not use path %v in a regexp: %v", path, err))
	}
	activity.Record(r.Context(), "API: Cache DELETE %v %+v", path, deleted)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(deleted); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal deleted keys for %v: %v", path, err))
	}
	return nil
}

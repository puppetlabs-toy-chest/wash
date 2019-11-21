package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:route DELETE /fs/delete delete deleteEntry
//
// Deletes the entry at the specified path.
//
// On success, returns a boolean that describes whether the delete was applied immediately
// or is pending.
//
//     Schemes: http
//
//     Responses:
//       200:
//       400: errorResp
//       404: errorResp
//       500: errorResp
var deleteHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}
	if !plugin.DeleteAction().IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.DeleteAction())
	}
	deleted, err := plugin.DeleteWithAnalytics(ctx, entry.(plugin.Deletable))
	if err != nil {
		return erroredActionResponse(path, plugin.DeleteAction(), err.Error())
	}
	activity.Record(ctx, "API: Delete %v %v", path, deleted)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(deleted); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal delete's result for %v: %v", path, err))
	}
	return nil
}

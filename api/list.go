package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:response
//nolint:deadcode,unused
type entryList struct {
	// in: body
	Entries []apitypes.Entry
}

// swagger:route GET /fs/list list listEntries
//
// Lists children of a path
//
// Returns a list of Entry objects describing children of the given path.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: entryList
//       400: errorResp
//       404: errorResp
//       500: errorResp
var listHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}

	if !plugin.ListAction().IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ListAction())
	}

	activity.Record(ctx, "API: List %v", path)
	group := entry.(plugin.Group)
	entries, err := plugin.CachedList(ctx, group)
	if err != nil {
		activity.Record(ctx, "API: List %v errored: %v", path, err)

		if cnameErr, ok := err.(plugin.DuplicateCNameErr); ok {
			return duplicateCNameResponse(cnameErr)
		}

		return erroredActionResponse(path, plugin.ListAction(), err.Error())
	}

	result := make([]apitypes.Entry, 0, len(entries))
	for _, entry := range entries {
		apiEntry := toAPIEntry(entry)
		apiEntry.Path = path + "/" + apiEntry.CName
		result = append(result, apiEntry)
	}
	activity.Record(ctx, "API: List %v %+v", path, result)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		activity.Record(ctx, "API: List marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal list results for %v: %v", path, err))
	}

	activity.Record(ctx, "API: List %v complete", path)
	return nil
}

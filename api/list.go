package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:route GET /fs/list list listEntries
//
// Lists children of a path
//
// Returns a list of ListEntry objects describing children of the given path.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: ListEntry
//       400: errorResp
//       404: errorResp
//       500: errorResp
var listHandler handler = func(w http.ResponseWriter, r *http.Request, p params) *errorResponse {
	path := p.Path
	ctx := r.Context()
	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.ListAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ListAction)
	}

	journal.Record(ctx, "API: List %v", path)
	group := entry.(plugin.Group)
	entries, err := plugin.CachedList(ctx, group)
	if err != nil {
		journal.Record(ctx, "API: List %v errored: %v", path, err)

		if cnameErr, ok := err.(plugin.DuplicateCNameErr); ok {
			return duplicateCNameResponse(cnameErr)
		}

		return erroredActionResponse(path, plugin.ListAction, err.Error())
	}

	info := func(entry plugin.Entry) apitypes.ListEntry {
		result := apitypes.ListEntry{
			Path:    plugin.Path(entry),
			Name:    entry.Name(),
			CName:   plugin.CName(entry),
			Actions: plugin.SupportedActionsOf(entry),
			Errors:  make(map[string]*apitypes.ErrorObj),
		}

		attr, err := plugin.Attr(r.Context(), entry)
		if err != nil {
			result.Errors["attributes"] = newUnknownErrorObj(err)
		} else {
			result.Attributes = attr
		}

		return result
	}

	result := make([]apitypes.ListEntry, len(entries)+1)
	result[0] = info(group)

	for i, entry := range entries {
		result[i+1] = info(entry)
	}
	journal.Record(ctx, "API: List %v %+v", path, result)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		journal.Record(ctx, "API: List marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal list results for %v: %v", path, err))
	}

	journal.Record(ctx, "API: List %v complete", path)
	return nil
}

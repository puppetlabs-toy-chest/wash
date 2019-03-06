package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var listHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodGet {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodGet})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: List %v", path)

	ctx := r.Context()
	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.ListAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ListAction)
	}

	plugin.Log(ctx, "API: List %v", path)
	group := entry.(plugin.Group)
	groupID := toID(path)
	entries, err := plugin.CachedList(ctx, group, groupID)
	if err != nil {
		plugin.Log(ctx, "API: List %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ListAction, err.Error())
	}

	info := func(entry plugin.Entry, entryID string) apitypes.ListEntry {
		result := apitypes.ListEntry{
			Name:    entry.Name(),
			Actions: plugin.SupportedActionsOf(entry),
			Errors:  make(map[string]*apitypes.ErrorObj),
		}

		err := plugin.FillAttr(r.Context(), entry, entryID, &result.Attributes)
		if err != nil {
			result.Errors["attributes"] = newUnknownErrorObj(err)
		}

		return result
	}

	result := make([]apitypes.ListEntry, len(entries)+1)
	result[0] = info(group, groupID)
	result[0].Name = "."

	for i, entry := range entries {
		result[i+1] = info(entry, groupID+"/"+entry.Name())
	}
	plugin.Log(ctx, "API: List %v %+v", path, result)

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		plugin.Log(ctx, "API: List marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal list results for %v: %v", path, err))
	}

	plugin.Log(ctx, "API: List %v complete", path)
	return nil
}

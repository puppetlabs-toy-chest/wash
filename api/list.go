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

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	if !plugin.ListAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ListAction)
	}

	group := entry.(plugin.Group)
	groupID := toID(path)
	entries, err := plugin.CachedList(r.Context(), group, groupID)
	if err != nil {
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

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal list results for %v: %v", path, err))
	}

	return nil
}

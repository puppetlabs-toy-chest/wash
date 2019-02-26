package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ListEntry represents a single entry from the result of issuing a wash "list"
// request.
//
// TODO: We should put all the API-specific types in a separate package so that
// clients do not have to import everything in api only to use a small subset
// of its functionality (the response types).
type ListEntry struct {
	Actions    []string             `json:"actions"`
	Name       string               `json:"name"`
	Attributes plugin.Attributes    `json:"attributes"`
	Errors     map[string]*ErrorObj `json:"errors"`
}

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

	group, ok := entry.(plugin.Group)
	if !ok {
		return unsupportedActionResponse(path, listAction)
	}

	groupID := toID(path)
	entries, err := plugin.CachedLS(r.Context(), group, groupID)
	if err != nil {
		return erroredActionResponse(path, listAction, err.Error())
	}

	info := func(entry plugin.Entry, entryID string) ListEntry {
		result := ListEntry{
			Name:    entry.Name(),
			Actions: supportedActionsOf(entry),
			Errors:  make(map[string]*ErrorObj),
		}

		err := plugin.FillAttr(r.Context(), entry, entryID, &result.Attributes)
		if err != nil {
			result.Errors["attributes"] = newUnknownErrorObj(err)
		}

		return result
	}

	result := make([]ListEntry, len(entries)+1)
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

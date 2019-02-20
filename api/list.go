package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
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

	group, ok := entry.(plugin.Group)
	if !ok {
		return unsupportedActionResponse(path, listAction)
	}

	entries, err := plugin.CachedLS(group, toID(path), r.Context())
	if err != nil {
		return erroredActionResponse(path, listAction, err.Error())
	}

	info := func(entry plugin.Entry) map[string]interface{} {
		result := map[string]interface{}{
			"name":    entry.Name(),
			"actions": supportedActionsOf(entry),
		}

		// TODO: use the FUSE logic for filling Attr. Not doing it yet because it overlaps
		// with in-progress caching work.
		if file, ok := entry.(plugin.File); ok {
			result["attributes"] = file.Attr()
		}

		return result
	}

	result := make([]map[string]interface{}, len(entries)+1)
	result[0] = info(group)
	result[0]["name"] = "."

	for i, entry := range entries {
		result[i+1] = info(entry)
	}

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal list results for %v: %v", path, err))
	}

	return nil
}

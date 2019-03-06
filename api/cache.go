package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var cacheHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodDelete {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodDelete})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Cache DELETE %v", path)

	ctx := r.Context()
	journal.Record(ctx, "API: Cache DELETE %v", path)
	deleted, err := plugin.ClearCacheFor(path)
	if err != nil {
		journal.Record(ctx, "API: Cache DELETE flush cache errored constructing regexp from %v: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not use path %v in a regexp: %v", path, err))
	}

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(deleted); err != nil {
		journal.Record(ctx, "API: Cache DELETE marshalling %v errored: %v", path, err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal deleted keys for %v: %v", path, err))
	}

	journal.Record(ctx, "API: Cache DELETE %v complete: %v", path, deleted)
	return nil
}

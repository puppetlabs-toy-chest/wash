package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/plugin"
)

var schemaHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}
	jsonEncoder := json.NewEncoder(w)
	if err := jsonEncoder.Encode(plugin.Schema(entry)); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal schema for %v: %v", path, err))
	}
	return nil
}

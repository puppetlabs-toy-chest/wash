package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var schemaHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}
	jsonEncoder := json.NewEncoder(w)
	s := entry.Schema()
	s.FillChildren()
	if err := jsonEncoder.Encode(s); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal schema for %v: %v", path, err))
	}
	return nil
}

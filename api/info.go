package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// swagger:route GET /fs/info info entryInfo
//
// Info about entry at path
//
// Returns an Entry object describing the given path.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: Entry
//       400: errorResp
//       404: errorResp
//       500: errorResp
var infoHandler = handler{fn: func(w http.ResponseWriter, r *http.Request) *errorResponse {
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}

	jsonEncoder := json.NewEncoder(w)
	// TODO: Include the entry's full metadata?
	apiEntry := toAPIEntry(entry)
	apiEntry.Path = path
	if err := jsonEncoder.Encode(&apiEntry); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal %v: %v", path, err))
	}
	return nil
}}

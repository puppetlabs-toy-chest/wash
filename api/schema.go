package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:response
//nolint:deadcode,unused
type schemaResponse struct {
	// in: body
	Schemas map[string]apitypes.EntrySchema
}

// swagger:route GET /fs/schema schema entrySchema
//
// Schema for an entry at path
//
// Returns a map of Type IDs to EntrySchema objects describing the plugin schema starting at the
// given path. The first key in the map corresponds to the path's schema.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: schemaResponse
//       400: errorResp
//       404: errorResp
//       500: errorResp
var schemaHandler = handler{fn: func(w http.ResponseWriter, r *http.Request) *errorResponse {
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}
	jsonEncoder := json.NewEncoder(w)
	schema, err := plugin.Schema(entry)
	if err != nil {
		return unknownErrorResponse(err)
	}
	apiEntrySchema := toAPIEntrySchema(schema)
	if err := jsonEncoder.Encode(apiEntrySchema); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal schema for %v: %v", path, err))
	}
	return nil
}}

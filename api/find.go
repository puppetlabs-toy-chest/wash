package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:route GET /fs/find find listEntries
//
// Recursively descends the given path returning it and its children.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: entryList
//       400: errorResp
//       404: errorResp
//       500: errorResp
var findHandler = handler{fn: func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}

	if !plugin.ListAction().IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ListAction())
	}

	minDepth, hasMinDepth, errResp := getIntParam(r.URL, "mindepth")
	if errResp != nil {
		return errResp
	}
	maxDepth, hasMaxDepth, errResp := getIntParam(r.URL, "maxdepth")
	if errResp != nil {
		return errResp
	}
	fullMeta, errResp := getBoolParam(r.URL, "fullmeta")
	if errResp != nil {
		return errResp
	}
	var rawQuery interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawQuery); err != nil {
		if err != io.EOF {
			return badRequestResponse(fmt.Sprintf("could not decode the RQL query: %v", err))
		}
		rawQuery = true
	}
	query := ast.Query()
	if err := query.Unmarshal(rawQuery); err != nil {
		return badRequestResponse(fmt.Sprintf("could not decode the RQL query: %v", err))
	}

	opts := rql.NewOptions()
	opts.Fullmeta = fullMeta
	if hasMinDepth {
		opts.Mindepth = minDepth
	}
	if hasMaxDepth {
		opts.Maxdepth = maxDepth
	}

	rqlEntries, err := rql.Find(ctx, entry, query, opts)
	if err != nil {
		return unknownErrorResponse(err)
	}

	result := []apitypes.Entry{}
	for _, rqlEntry := range rqlEntries {
		apiEntry := rqlEntry.Entry
		// Make sure all paths are absolute paths
		apiEntry.Path = path + "/" + apiEntry.Path
		result = append(result, apiEntry)
	}

	activity.Record(ctx, "API: Find %v %v items", path, len(result))

	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal find results for %v: %v", path, err))
	}
	return nil
}}

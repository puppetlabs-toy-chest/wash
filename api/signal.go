package api

import (
	"encoding/json"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:route POST /fs/signal signal signalEntry
//
// Sends a signal to the entry at the specified path.
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
//       200:
//       400: errorResp
//       404: errorResp
//       500: errorResp
var signalHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}

	if !plugin.SignalAction().IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.SignalAction())
	}

	if r.Body == nil {
		return badActionRequestResponse(path, plugin.SignalAction(), "Please send a JSON request body")
	}

	var body apitypes.SignalBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return badActionRequestResponse(path, plugin.SignalAction(), err.Error())
	}

	if err := plugin.SignalWithAnalytics(ctx, entry.(plugin.Signalable), body.Signal); err != nil {
		if plugin.IsInvalidInputErr(err) {
			return badActionRequestResponse(path, plugin.SignalAction(), err.Error())
		}
		return erroredActionResponse(path, plugin.SignalAction(), err.Error())
	}

	activity.Record(ctx, "API: Signal %v %v", path, body.Signal)
	return nil
}

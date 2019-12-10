package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/wash/analytics"
	apitypes "github.com/puppetlabs/wash/api/types"
	log "github.com/sirupsen/logrus"
)

// swagger:route POST /analytics/screenview
//
// Submits a screenview to Google Analytics
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
//       404: errorResp
var screenviewHandler = handler{logOnly: true, fn: func(w http.ResponseWriter, r *http.Request) *errorResponse {
	var body apitypes.ScreenviewBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return badRequestResponse(fmt.Sprintf("Error unmarshalling the request body: %v", err))
	}
	ctx := r.Context()
	err := analytics.GetClient(ctx).Screenview(body.Name, body.Params)
	if err != nil {
		log.Info(err)
		return badRequestResponse(err.Error())
	}
	return nil
}}

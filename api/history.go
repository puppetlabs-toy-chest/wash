package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/activity"
	log "github.com/sirupsen/logrus"
)

// swagger:response
//nolint:deadcode,unused
type journals struct {
	// in: body
	Journals []activity.Journal
}

// swagger:route GET /history history retrieveHistory
//
// Get command history
//
// Get a list of commands that have been run via 'wash' and when they were run.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: journals
//       400: errorResp
//       404: errorResp
//       500: errorResp
var historyHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	history := activity.History()

	w.WriteHeader(http.StatusOK)

	commands := make([]string, len(history))
	for i, item := range history {
		// TODO: send structured data instead of a pre-formatted string
		commands[i] = item.Start.Format("2006-01-02 15:04") + "  " + item.Description
	}
	jsonEncoder := json.NewEncoder(w)
	if err := jsonEncoder.Encode(&commands); err != nil {
		log.Warnf("Unable to encode history: %v", err)
		return unknownErrorResponse(fmt.Errorf("Could not marshal %v: %v", history, err))
	}
	return nil
}

// swagger:route GET /history/{id} journal getJournal
//
// Get logs for a particular entry in history
//
// Get the logs related to a particular command run via 'wash', requested by
// index within its activity history.
//
//     Produces:
//     - application/json
//     - application/octet-stream
//
//     Schemes: http
//
//     Responses:
//       200: octetResponse
//       400: errorResp
//       404: errorResp
//       500: errorResp
var historyEntryHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	history := activity.History()
	index := mux.Vars(r)["index"]

	idx, err := strconv.Atoi(index)
	if err != nil || idx < 0 || idx >= len(history) {
		if err == nil {
			err = fmt.Errorf("index out of bounds")
		}
		return outOfBoundsRequest(len(history), err.Error())
	}

	journal := history[idx]
	rdr, err := journal.Open()
	if err != nil {
		return journalUnavailableResponse(journal.String(), err.Error())
	}
	defer func() {
		if err := rdr.Close(); err != nil {
			log.Infof("Failed to close journal %v: %v", journal, err)
		}
	}()

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rdr); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not read journal %v: %v", journal, err))
	}
	return nil
}

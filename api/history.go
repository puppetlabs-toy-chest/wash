package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/activity"
	apitypes "github.com/puppetlabs/wash/api/types"
)

// swagger:parameters retrieveHistory getJournal
//nolint:deadcode,unused
type historyParams struct {
	// stream updates when true
	//
	// in: query
	Follow bool
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
//       200: HistoryResponse
//       400: errorResp
//       404: errorResp
//       500: errorResp
var historyHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	follow, err := getBoolParam(r.URL, "follow")
	if err != nil {
		return err
	}

	var enc *json.Encoder
	if follow {
		// Ensure every write is a flush.
		f, ok := w.(flushableWriter)
		if !ok {
			return unknownErrorResponse(fmt.Errorf("Cannot stream history, response handler does not support flushing"))
		}
		enc = json.NewEncoder(&streamableResponseWriter{f})
	} else {
		enc = json.NewEncoder(w)
	}

	history := activity.History()
	if err := writeHistory(r.Context(), enc, history); err != nil {
		return err
	}

	if follow {
		last := len(history)
		for {
			// Continue sending updates
			select {
			case <-r.Context().Done():
				return nil
			case <-time.After(1 * time.Second):
				// Retry
			}

			history = activity.History()
			if len(history) > last {
				if err := writeHistory(r.Context(), enc, history[last:]); err != nil {
					return err
				}
				last = len(history)
			}
		}
	}
	return nil
}

func writeHistory(ctx context.Context, enc *json.Encoder, history []activity.Journal) *errorResponse {
	var act apitypes.Activity
	for _, item := range history {
		act.Description = item.Description
		act.Start = item.Start()
		if err := enc.Encode(&act); err != nil {
			return unknownErrorResponse(fmt.Errorf("Could not marshal %v: %v", history, err))
		}
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

	follow, errResp := getBoolParam(r.URL, "follow")
	if errResp != nil {
		return errResp
	}

	journal := history[idx]

	if follow {
		// Ensure every write is a flush.
		f, ok := w.(flushableWriter)
		if !ok {
			return unknownErrorResponse(fmt.Errorf("Cannot stream history, response handler does not support flushing"))
		}

		rdr, err := journal.Tail()
		if err != nil {
			return journalUnavailableResponse(journal.String(), err.Error())
		}

		// Ensure the reader is closed when context stops.
		go func() {
			<-r.Context().Done()
			rdr.Cleanup()
			activity.Record(r.Context(), "API: Journal %v closed by completed context: %v", journal, rdr.Stop())
		}()

		// Do an initial flush to send the header.
		w.WriteHeader(http.StatusOK)
		f.Flush()

		for line := range rdr.Lines {
			if line.Err != nil {
				return unknownErrorResponse(line.Err)
			}
			if _, err := fmt.Fprintln(f, line.Text); err != nil {
				return unknownErrorResponse(err)
			}
			f.Flush()
		}
	} else {
		rdr, err := journal.Open()
		if err != nil {
			return journalUnavailableResponse(journal.String(), err.Error())
		}

		defer func() {
			activity.Record(r.Context(), "API: Journal %v closed by completed context: %v", journal, rdr.Close())
		}()

		if _, err := io.Copy(w, rdr); err != nil {
			return unknownErrorResponse(fmt.Errorf("Could not read journal %v: %v", journal, err))
		}
	}
	return nil
}

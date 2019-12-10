package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// swagger:route GET /fs/stream stream streamUpdates
//
// Stream updates
//
// Get a stream of new updates to the specified entry.
//
//     Produces:
//     - application/json
//     - application/octet-stream
//
//     Schemes: http
//
//     Responses:
//       200: octetResponse
//       404: errorResp
//       500: errorResp
var streamHandler = handler{fn: func(w http.ResponseWriter, r *http.Request) *errorResponse {
	entry, path, errResp := getEntryFromRequest(r)
	if errResp != nil {
		return errResp
	}

	if !plugin.StreamAction().IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.StreamAction())
	}

	f, ok := w.(flushableWriter)
	if !ok {
		return unknownErrorResponse(fmt.Errorf("Cannot stream %v, response handler does not support flushing", path))
	}

	ctx := r.Context()
	rdr, err := plugin.StreamWithAnalytics(ctx, entry.(plugin.Streamable))

	if err != nil {
		return erroredActionResponse(path, plugin.StreamAction(), err.Error())
	}
	activity.Record(ctx, "API: Streaming %v", path)

	// Do an initial flush to send the header.
	w.WriteHeader(http.StatusOK)
	f.Flush()

	// Ensure it's closed when the context is cancelled.
	go func() {
		<-ctx.Done()
		activity.Record(ctx, "API: Stream %v closed by completed context: %v", path, rdr.Close())
	}()

	// Ensure every write is a flush with streamableResponseWriter.
	if _, err := io.Copy(&streamableResponseWriter{f}, rdr); err != nil {
		// Common for copy to error when the caller closes the connection.
		activity.Record(ctx, "API: Streaming %v errored: %v", path, err)
	}
	return nil
}}

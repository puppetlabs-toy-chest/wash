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
var streamHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
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

	rdr, err := entry.(plugin.Streamable).Stream(ctx)

	if err != nil {
		return erroredActionResponse(path, plugin.StreamAction(), err.Error())
	}
	activity.Record(ctx, "API: Streaming %v", path)

	w.WriteHeader(http.StatusOK)
	// Ensure every write is a flush, and do an initial flush to send the header.
	wf := &streamableResponseWriter{f}
	f.Flush()

	// Ensure it's closed when the context is cancelled.
	go func() {
		<-r.Context().Done()
		activity.Record(ctx, "API: Stream %v closed by completed context: %v", path, rdr.Close())
	}()

	if _, err := io.Copy(wf, rdr); err != nil {
		// Common for copy to error when the caller closes the connection.
		activity.Record(ctx, "API: Streaming %v errored: %v", path, err)
	}
	return nil
}

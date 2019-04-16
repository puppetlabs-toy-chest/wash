package api

import (
	"io"
	"net/http"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// swagger:route GET /fs/read read readContent
//
// Read content
//
// Read content from the specified entry.
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
var readHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(ctx, r)
	if errResp != nil {
		return errResp
	}

	if !plugin.ReadAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ReadAction)
	}

	activity.Record(ctx, "API: Read %v", path)
	content, err := plugin.CachedOpen(ctx, entry.(plugin.Readable))

	if err != nil {
		activity.Record(ctx, "API: Read %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}
	activity.Record(ctx, "API: Reading %v", path)

	w.WriteHeader(http.StatusOK)
	n, err := io.Copy(w, io.NewSectionReader(content, 0, content.Size()))
	if n != content.Size() {
		log.Warnf("Read incomplete %v/%v", n, content.Size())
		activity.Record(ctx, "API: Reading %v incomplete: %v/%v", path, n, content.Size())
	}
	if err != nil {
		activity.Record(ctx, "API: Reading %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}

	activity.Record(ctx, "API: Reading %v complete", path)
	return nil
}

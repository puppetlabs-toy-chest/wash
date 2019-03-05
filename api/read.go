package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var readHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodGet {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodGet})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Read %v", path)

	jnl := journal.NamedJournal{ID: r.FormValue(apitypes.JournalID)}
	ctx := context.WithValue(r.Context(), plugin.Journal, jnl)

	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	if !plugin.ReadAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ReadAction)
	}

	jnl.Log("API: Read %v", path)
	content, err := plugin.CachedOpen(ctx, entry.(plugin.Readable), toID(path))

	if err != nil {
		jnl.Log("API: Read %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}
	jnl.Log("API: Reading %v", path)

	w.WriteHeader(http.StatusOK)
	n, err := io.Copy(w, io.NewSectionReader(content, 0, content.Size()))
	if n != content.Size() {
		log.Warnf("Read incomplete %v/%v", n, content.Size())
		jnl.Log("API: Reading %v incomplete: %v/%v", path, n, content.Size())
	}
	if err != nil {
		jnl.Log("API: Reading %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ReadAction, err.Error())
	}

	jnl.Log("API: Reading %v complete", path)
	return nil
}

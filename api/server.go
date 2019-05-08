package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/activity"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

type key int

const (
	pluginRegistryKey key = iota
	mountpointKey
)

// swagger:parameters cacheDelete listEntries entryInfo executeCommand getMetadata readContent streamUpdates
//nolint:deadcode,unused
type params struct {
	// uniquely identifies an entry
	//
	// in: query
	Path string
}

// swagger:response
//nolint:deadcode,unused
type octetResponse struct {
	// in: body
	Reader io.Reader
}

type handler func(http.ResponseWriter, *http.Request) *errorResponse

func (handle handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	activity.Record(r.Context(), "API: %v %v", r.Method, r.URL)

	if err := handle(w, r); err != nil {
		w.WriteHeader(err.statusCode)

		// NOTE: Do not set these headers in the middleware because not
		// all API calls are guaranteed to return JSON responses (e.g. like
		// stream and read)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if _, err := fmt.Fprintln(w, err.Error()); err != nil {
			log.Warnf("API: Failed writing error response: %v", err)
		}
	}
}

// StartAPI starts the api. It returns three values:
//   1. A channel to initiate the shutdown (stopCh). stopCh accepts a Context object
//      that is used to cancel a stalled shutdown.
//
//   2. A read-only channel that signals whether the server was shutdown.
//
//   3. An error object
func StartAPI(registry *plugin.Registry, mountpoint string, socketPath string) (chan<- context.Context, <-chan struct{}, error) {
	log.Infof("API: Listening at %s", socketPath)

	if _, err := os.Stat(socketPath); err == nil {
		// Socket already exists, so nuke it and recreate it
		log.Infof("API: Cleaning up old socket")
		if err := os.Remove(socketPath); err != nil {
			return nil, nil, err
		}
	} else {
		// Ensure the parent directory for the socket path exists
		if err := os.MkdirAll(filepath.Dir(socketPath), 0750); err != nil {
			return nil, nil, err
		}
	}

	server, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, err
	}

	prepareContextMiddleWare := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newctx := context.WithValue(r.Context(), pluginRegistryKey, registry)
			newctx = context.WithValue(newctx, mountpointKey, mountpoint)
			journal := activity.NewJournal(
				r.Header.Get(apitypes.JournalIDHeader),
				r.Header.Get(apitypes.JournalDescHeader),
			)
			newctx = context.WithValue(newctx, activity.JournalKey, journal)

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r.WithContext(newctx))
		})
	}

	r := mux.NewRouter()

	r.Handle("/fs/info", infoHandler).Methods(http.MethodGet)
	r.Handle("/fs/list", listHandler).Methods(http.MethodGet)
	r.Handle("/fs/metadata", metadataHandler).Methods(http.MethodGet)
	r.Handle("/fs/read", readHandler).Methods(http.MethodGet)
	r.Handle("/fs/stream", streamHandler).Methods(http.MethodGet)
	r.Handle("/fs/exec", execHandler).Methods(http.MethodPost)
	r.Handle("/cache", cacheHandler).Methods(http.MethodDelete)
	r.Handle("/history", historyHandler).Methods(http.MethodGet)
	r.Handle("/history/{index:[0-9]+}", historyEntryHandler).Methods(http.MethodGet)

	r.Use(prepareContextMiddleWare)

	httpServer := http.Server{Handler: r}

	// Start the server
	serverStoppedCh := make(chan struct{})
	go func() {
		defer close(serverStoppedCh)

		err := httpServer.Serve(server)
		if err != nil && err != http.ErrServerClosed {
			log.Warnf("API: %v", err)
		}

		log.Infof("API: Server was shut down")
	}()

	stopCh := make(chan context.Context)
	go func() {
		ctx := <-stopCh

		log.Infof("API: Shutting down the server")
		err := httpServer.Shutdown(ctx)
		if err != nil {
			log.Warnf("API: Shutdown failed: %v", err)
		}

		<-serverStoppedCh
	}()

	return stopCh, serverStoppedCh, nil
}

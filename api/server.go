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
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

type key int

const pluginRegistryKey key = iota

// swagger:parameters cacheDelete listEntries executeCommand getMetadata readContent streamUpdates
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

type handler func(http.ResponseWriter, *http.Request, params) *errorResponse

func (handle handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("API: %v %v", r.Method, r.URL)

	paths := r.URL.Query()["path"]
	if len(paths) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Request must include one 'path' query parameter")
		return
	}

	if err := handle(w, r, params{Path: paths[0]}); err != nil {
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
func StartAPI(registry *plugin.Registry, socketPath string) (chan<- context.Context, <-chan struct{}, error) {
	log.Infof("API: started")

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

	addPluginRegistryAndJournalIDMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newctx := context.WithValue(r.Context(), pluginRegistryKey, registry)
			newctx = context.WithValue(newctx, journal.Key, r.Header.Get(apitypes.JournalIDHeader))

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r.WithContext(newctx))
		})
	}

	r := mux.NewRouter()

	r.Handle("/fs/list", listHandler).Methods(http.MethodGet)
	r.Handle("/fs/metadata", metadataHandler).Methods(http.MethodGet)
	r.Handle("/fs/read", readHandler).Methods(http.MethodGet)
	r.Handle("/fs/stream", streamHandler).Methods(http.MethodGet)
	r.Handle("/fs/exec", execHandler).Methods(http.MethodPost)
	r.Handle("/cache", cacheHandler).Methods(http.MethodDelete)

	r.Use(addPluginRegistryAndJournalIDMiddleware)

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
		defer close(stopCh)
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

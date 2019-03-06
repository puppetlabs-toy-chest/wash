package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

type key int

const pluginRegistryKey key = iota

type handler func(http.ResponseWriter, *http.Request) *errorResponse

func (handle handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
func StartAPI(registry *plugin.Registry, socketPath string) (chan<- context.Context, <-chan struct{}, error) {
	log.Infof("API: started")

	if _, err := os.Stat(socketPath); err == nil {
		// Socket already exists, so nuke it and recreate it
		log.Infof("API: Cleaning up old socket")
		if err := os.Remove(socketPath); err != nil {
			log.Warnf("API: %v", err)
			return nil, nil, err
		}
	}

	server, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Warnf("API: %v", err)
		return nil, nil, err
	}

	addPluginRegistryAndJournalMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newctx := context.WithValue(r.Context(), pluginRegistryKey, registry)
			newctx = context.WithValue(newctx, plugin.Journal, r.Header.Get(apitypes.JournalID))

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r.WithContext(newctx))
		})
	}

	r := mux.NewRouter()

	r.Handle("/fs/list/{path:.+}", listHandler)
	r.Handle("/fs/metadata/{path:.+}", metadataHandler)
	r.Handle("/fs/read/{path:.+}", readHandler)
	r.Handle("/fs/stream/{path:.+}", streamHandler)
	r.Handle("/fs/exec/{path:.+}", execHandler)

	r.Use(addPluginRegistryAndJournalMiddleware)

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

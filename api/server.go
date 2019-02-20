package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
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

		fmt.Fprintln(w, err.Error())
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

	addPluginRegistryMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newr := r.WithContext(context.WithValue(r.Context(), pluginRegistryKey, registry))

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, newr)
		})
	}

	r := mux.NewRouter()

	r.Handle("/fs/list/{path:.+}", listHandler)
	r.Handle("/fs/metadata/{path:.+}", metadataHandler)
	r.Handle("/fs/read/{path:.+}", readHandler)
	r.Handle("/fs/stream/{path:.+}", streamHandler)

	r.Use(addPluginRegistryMiddleware)

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

func findEntry(ctx context.Context, root plugin.Entry, segments []string) (plugin.Entry, *errorResponse) {
	path := strings.Join(segments, "/")

	curEntry := root
	curEntryID := "/" + root.Name()

	visitedSegments := make([]string, 0, cap(segments))
	for _, segment := range segments {
		switch curGroup := curEntry.(type) {
		case plugin.Group:
			// Get the entries via. LS()
			entries, err := plugin.CachedLS(curGroup, curEntryID, ctx)
			if err != nil {
				return nil, entryNotFoundResponse(path, err.Error())
			}

			// Search for the specific entry
			var found bool
			for _, entry := range entries {
				if entry.Name() == segment {
					found = true

					curEntry = entry
					curEntryID += "/" + segment
					visitedSegments = append(visitedSegments, segment)

					break
				}
			}
			if !found {
				reason := fmt.Sprintf("The %v entry does not exist", segment)
				if len(visitedSegments) != 0 {
					reason += fmt.Sprintf(" the %v group", strings.Join(visitedSegments, "/"))
				}

				return nil, entryNotFoundResponse(path, reason)
			}
		default:
			reason := fmt.Sprintf("The entry %v is not a group", strings.Join(visitedSegments, "/"))
			return nil, entryNotFoundResponse(path, reason)
		}
	}

	return curEntry, nil
}

func getEntryFromPath(ctx context.Context, path string) (plugin.Entry, *errorResponse) {
	if path == "" {
		panic("path should never be empty")
	}

	// Don't interpret trailing slash as a new segment
	path = strings.TrimSuffix(path, "/")
	// Split into plugin name and an optional list of segments.
	segments := strings.Split(path, "/")
	pluginName := segments[0]
	segments = segments[1:]

	// Get the registry from context (added by registry middleware).
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	root, ok := registry.Plugins[pluginName]
	if !ok {
		return nil, pluginDoesNotExistResponse(pluginName)
	}

	return findEntry(ctx, root, segments)
}

func toID(path string) string {
	return "/" + strings.TrimSuffix(path, "/")
}

package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

// ApiHandler is the API handler
type ApiHandler struct {
	pluginRegistry *plugin.Registry
}

type key int

const pluginRegistryKey key = iota

// StartAPI starts the api
func StartAPI(registry *plugin.Registry, socketPath string) error {
	log.Printf("API: started")

	if _, err := os.Stat(socketPath); err == nil {
		// Socket already exists, so nuke it and recreate it
		log.Printf("API: Cleaning up old socket")
		if err := os.Remove(socketPath); err != nil {
			log.Warnf("API: %v", err)
			return err
		}
	}

	server, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Warnf("API: %v", err)
		return err
	}

	addPluginRegistryMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newr := r.WithContext(context.WithValue(r.Context(), pluginRegistryKey, registry))

			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, newr)
		})
	}

	r := mux.NewRouter()
	r.Handle("/fs/{plugin}/{path:.*}", ApiHandler{pluginRegistry: registry})
	r.Use(addPluginRegistryMiddleware)
	return http.Serve(server, r)
}

func segmentsFromRawURLPath(rawPath string) ([]string, error) {
	segments := strings.Split(rawPath, "/")
	for i, rawSegment := range segments {
		segment, err := url.PathUnescape(rawSegment)
		if err != nil {
			return nil, err
		}

		segments[i] = segment
	}

	return segments, nil
}

func (handler ApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["plugin"]
	rawPath := vars["path"]

	ctx := r.Context()
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	root, ok := registry.Plugins[pluginName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Plugin %v does not exist\n", pluginName)
		return
	}

	segments, err := segmentsFromRawURLPath(rawPath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "The path %v is malformed: %v\n", strings.Join(segments, "/"), err)
		return
	}

	entry, err := plugin.FindEntryByPath(ctx, root, segments)
	if err != nil {
		// TODO: Make the error structured so we can distinguish between
		// a NotFound vs. bad path vs. other stuff
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Entry not found: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Found entry %v\n", entry.Name())
}

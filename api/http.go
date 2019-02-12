package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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
	r.Handle("/fs/{path:.*}", ApiHandler{pluginRegistry: registry})
	r.Use(addPluginRegistryMiddleware)
	return http.Serve(server, r)
}

// Query parameter ?op=metadata
func (handler ApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	// Get the operation for early validation
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Could not parse request parameters: %v\n", err)
		return
	}

	op := r.Form.Get("op")
	if op == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Must provide the op query parameter\n")
		return
	}

	if op != "metadata" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Operation %v is not supported\n", op)
		return
	}

	segments := strings.Split(path, "/")
	pluginName := segments[0]
	segments = segments[1:]

	ctx := r.Context()
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	root, ok := registry.Plugins[pluginName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Plugin %v does not exist\n", pluginName)
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

	switch op {
	// TODO: Make "metadata" constant at some point
	case "metadata":
		if resource, ok := entry.(plugin.Resource); ok {
			metadata, err := resource.Metadata(ctx)

			// TODO: Definitely figure out the error handling at some
			// point
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Could not get metadata for %v: %v\n", path, err)
				return
			}

			w.WriteHeader(http.StatusOK)
			jsonEncoder := json.NewEncoder(w)
			if err = jsonEncoder.Encode(metadata); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Could not marshal metadata for %v: %v\n", path, err)
				return
			}

			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Entry %v does not support the %v operation\n", path, op)
	return
}

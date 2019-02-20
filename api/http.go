package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	entry, err := plugin.FindEntryByPath(ctx, root, segments)
	if err != nil {
		return nil, entryNotFoundResponse(path, err.Error())
	}

	return entry, nil
}

var listHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	path := mux.Vars(r)["path"]
	log.Infof("API: List %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	group, ok := entry.(plugin.Group)
	if !ok {
		return unsupportedActionResponse(path, listAction)
	}

	entries, err := group.LS(r.Context())
	if err != nil {
		return erroredActionResponse(path, listAction, err.Error())
	}

	info := func(entry plugin.Entry) map[string]interface{} {
		result := map[string]interface{}{
			"name":    entry.Name(),
			"actions": supportedActionsOf(entry),
		}

		if file, ok := entry.(plugin.File); ok {
			result["attributes"] = file.Attr()
		}

		return result
	}

	result := make([]map[string]interface{}, len(entries)+1)
	result[0] = info(group)
	result[0]["name"] = "."

	for i, entry := range entries {
		result[i+1] = info(entry)
	}

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(result); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal list results for %v: %v", path, err))
	}

	return nil
}

var metadataHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	path := mux.Vars(r)["path"]
	log.Infof("API: Metadata %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	resource, ok := entry.(plugin.Resource)
	if !ok {
		return unsupportedActionResponse(path, metadataAction)
	}

	metadata, err := resource.Metadata(r.Context())

	if err != nil {
		return erroredActionResponse(path, metadataAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		return unknownErrorResponse(fmt.Errorf("Could not marshal metadata for %v: %v", path, err))
	}

	return nil
}

var readHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	path := mux.Vars(r)["path"]
	log.Infof("API: Read %v", path)

	entry, errResp := getEntryFromPath(r.Context(), path)
	if errResp != nil {
		return errResp
	}

	readable, ok := entry.(plugin.Readable)
	if !ok {
		return unsupportedActionResponse(path, readAction)
	}

	content, err := readable.Open(r.Context())

	if err != nil {
		return erroredActionResponse(path, readAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	if rdr, ok := content.(io.Reader); ok {
		io.Copy(w, rdr)
	} else {
		io.Copy(w, io.NewSectionReader(content, 0, content.Size()))
	}

	return nil
}

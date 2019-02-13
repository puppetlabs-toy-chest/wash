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

	r.HandleFunc("/fs/list/{path:.+}", listHandler)
	r.HandleFunc("/fs/metadata/{path:.+}", metadataHandler)

	r.Use(addPluginRegistryMiddleware)
	return http.Serve(server, r)
}

func getEntryFromRequest(r *http.Request) (plugin.Entry, string, error) {
	vars := mux.Vars(r)
	path := vars["path"]
	if path == "" {
		panic("path should never be empty")
	}

	// What happens if segments is empty
	segments := strings.Split(path, "/")
	pluginName := segments[0]
	segments = segments[1:]

	ctx := r.Context()
	registry := ctx.Value(pluginRegistryKey).(*plugin.Registry)
	root, ok := registry.Plugins[pluginName]
	if !ok {
		return nil, path, fmt.Errorf("Plugin %v does not exist", pluginName)
	}

	entry, err := plugin.FindEntryByPath(ctx, root, segments)
	if err != nil {
		return nil, path, fmt.Errorf("Entry not found: %v", err)
	}

	return entry, path, nil
}

func supportedCommands(entry plugin.Entry) []string {
	commands := make([]string, 0)

	if _, ok := entry.(plugin.Group); ok {
		commands = append(commands, plugin.ListCommand)
	}

	if _, ok := entry.(plugin.Resource); ok {
		commands = append(commands, plugin.MetadataCommand)
	}

	if _, ok := entry.(plugin.Readable); ok {
		commands = append(commands, plugin.ReadCommand)
	}

	return commands
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	entry, path, err := getEntryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	group, ok := entry.(plugin.Group)
	if !ok {
		http.Error(w, fmt.Sprintf("Entry %v does not support the list command", path), http.StatusNotFound)
		return
	}

	entries, err := group.LS(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not list the entries for %v: %v", path, err), http.StatusInternalServerError)
		return
	}

	info := func(entry plugin.Entry) map[string]interface{} {
		result := map[string]interface{}{
			"name":     entry.Name(),
			"commands": supportedCommands(entry),
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
		http.Error(w, fmt.Sprintf("Could not marshal list results for %v: %v", path, err), http.StatusInternalServerError)
		return
	}
}

func metadataHandler(w http.ResponseWriter, r *http.Request) {
	entry, path, err := getEntryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resource, ok := entry.(plugin.Resource)
	if !ok {
		http.Error(w, fmt.Sprintf("Entry %v does not support the metadata command", path), http.StatusNotFound)
		return
	}

	metadata, err := resource.Metadata(r.Context())

	// TODO: Definitely figure out the error handling at some
	// point
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get metadata for %v: %v\n", path, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	jsonEncoder := json.NewEncoder(w)
	if err = jsonEncoder.Encode(metadata); err != nil {
		http.Error(w, fmt.Sprintf("Could not marshal metadata for %v: %v\n", path, err), http.StatusInternalServerError)
		return
	}
}

package plugin

import (
	"context"
)

// Registry represents the plugin registry. It is also Wash's root.
type Registry struct {
	EntryBase
	plugins     map[string]Root
	pluginRoots []Entry
}

// NewRegistry creates a new plugin registry object
func NewRegistry() *Registry {
	r := &Registry{
		EntryBase: NewEntry("/"),
		plugins:   make(map[string]Root),
	}

	// Set the registry's ID to the empty string. This way,
	// CachedList sets the cache IDs of the Plugin roots to
	// "/<root_name>" (e.g. "/docker", "/kubernetes", "/aws"),
	// and all other IDs are correctly set to <parent_id> + "/" + <name>.
	r.CacheConfig().id = ""
	r.CacheConfig().TurnOffCaching()

	return r
}

// Plugins returns a map of the currently registered
// plugins
func (r *Registry) Plugins() map[string]Root {
	return r.plugins
}

// RegisterPlugin registers the given plugin
func (r *Registry) RegisterPlugin(name string, root Root) {
	r.plugins[name] = root
	r.pluginRoots = append(r.pluginRoots, root)
}

// List all of Wash's loaded plugins
func (r *Registry) List(ctx context.Context) ([]Entry, error) {
	return r.pluginRoots, nil
}

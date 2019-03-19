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
		EntryBase: newEntryBase("/"),
		plugins:   make(map[string]Root),
	}
	r.setID("/")
	r.TurnOffCaching()

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

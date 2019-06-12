package plugin

import (
	"context"
	"fmt"
	"regexp"
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
		EntryBase: NewEntryBase(),
		plugins:   make(map[string]Root),
	}
	r.SetName("/")
	r.setID("/")
	r.DisableDefaultCaching()

	return r
}

// Plugins returns a map of the currently registered
// plugins
func (r *Registry) Plugins() map[string]Root {
	return r.plugins
}

var pluginNameRegex = regexp.MustCompile("^[0-9a-zA-Z_-]+$")

// RegisterPlugin initializes the given plugin and adds it to the registry if
// initialization was successful.
func (r *Registry) RegisterPlugin(root Root, config map[string]interface{}) error {
	if err := root.Init(config); err != nil {
		return err
	}

	if !pluginNameRegex.MatchString(root.name()) {
		msg := fmt.Sprintf("r.RegisterPlugin: invalid plugin name %v. The plugin name must consist of alphanumeric characters, or a hyphen", root.name())
		panic(msg)
	}

	if _, ok := r.plugins[root.name()]; ok {
		msg := fmt.Sprintf("r.RegisterPlugin: the %v plugin's already been registered", root.name())
		panic(msg)
	}

	root.entryBase().IsSingleton()
	r.plugins[root.name()] = root
	r.pluginRoots = append(r.pluginRoots, root)
	return nil
}

// ChildSchemas returns the child schemas of the plugin registry
func (r *Registry) ChildSchemas() []EntrySchema {
	return ChildSchemas(r.pluginRoots...)
}

// List all of Wash's loaded plugins
func (r *Registry) List(ctx context.Context) ([]Entry, error) {
	return r.pluginRoots, nil
}

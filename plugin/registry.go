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
		EntryBase: NewEntry("/"),
		plugins:   make(map[string]Root),
	}
	r.setID("/")
	r.DisableDefaultCaching()
	r.isPluginRegistry = true

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

	r.plugins[root.name()] = root
	r.pluginRoots = append(r.pluginRoots, root)
	return nil
}

// ChildSchemas returns the child schemas of the plugin registry
func (r *Registry) ChildSchemas() []*EntrySchema {
	var childSchemas []*EntrySchema
	for _, root := range r.pluginRoots {
		s := root.Schema()
		if s == nil {
			// s doesn't have a schema, which means it's an external plugin.
			//
			// TODO: This makes it possible for core plugins to return nil
			// schemas, which shouldn't happen. Find a way to rectify this once
			// external plugin schemas are supported.
			continue
		}
		s.IsSingleton()
		if len(s.Label()) == 0 {
			s.SetLabel(CName(root))
		}
		childSchemas = append(childSchemas, root.Schema())
	}
	return childSchemas
}

// Schema returns the plugin registry's schema
func (r *Registry) Schema() *EntrySchema {
	return NewEntrySchema(r, "mountpoint").IsSingleton()
}

// List all of Wash's loaded plugins
func (r *Registry) List(ctx context.Context) ([]Entry, error) {
	return r.pluginRoots, nil
}

package plugin

import (
	"context"
	"fmt"
	"os"
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
	r.DisableDefaultCaching()

	return r
}

// Plugins returns a map of the currently registered
// plugins
func (r *Registry) Plugins() map[string]Root {
	return r.plugins
}

// RegisterPlugin initializes the given plugin and adds it to the registry if
// initialization was successful.
func (r *Registry) RegisterPlugin(root Root) error {
	if err := root.Init(); err != nil {
		return err
	}

	r.plugins[root.Name()] = root
	r.pluginRoots = append(r.pluginRoots, root)
	return nil
}

// ExternalPluginSpec represents an external plugin's specification.
type ExternalPluginSpec struct {
	Script string
}

// RegisterExternalPlugin initializes an external plugin from its spec and
// passes it to RegisterPlugin.
func (r *Registry) RegisterExternalPlugin(spec ExternalPluginSpec) error {
	fi, err := os.Stat(spec.Script)
	if err != nil {
		return err
	} else if !fi.Mode().IsRegular() {
		return fmt.Errorf("script %v is not a file", spec.Script)
	} else if fi.Mode().Perm()&0100 == 0 {
		return fmt.Errorf("script %v is not executable", spec.Script)
	}

	root := newExternalPluginRoot(spec.Script)
	return r.RegisterPlugin(root)
}

// List all of Wash's loaded plugins
func (r *Registry) List(ctx context.Context) ([]Entry, error) {
	return r.pluginRoots, nil
}

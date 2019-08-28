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
	registerPlugin := func() {
		r.plugins[root.name()] = root
		r.pluginRoots = append(r.pluginRoots, root)
	}

	if err := root.Init(config); err != nil {
		// Create a stubPluginRoot so that Wash users can see the plugin's
		// documentation via 'describe <plugin>'. This is important b/c the
		// plugin docs also include details on how to set it up. Note that
		// 'describe <plugin>' will not work for external plugins. This is because
		// the plugin documentation is contained in the root's description, and
		// the root's description is contained in the root's schema. Retrieving
		// an external plugin root's schema requires a successful Init invocation,
		// which is not the case here.
		root = newStubRoot(root)
		registerPlugin()
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

	registerPlugin()
	return nil
}

// ChildSchemas only makes sense for core plugin roots
func (r *Registry) ChildSchemas() []*EntrySchema {
	return nil
}

// Schema only makes sense for core plugin roots
func (r *Registry) Schema() *EntrySchema {
	return nil
}

// List all of Wash's loaded plugins
func (r *Registry) List(ctx context.Context) ([]Entry, error) {
	return r.pluginRoots, nil
}

type stubRoot struct {
	EntryBase
	pluginDocumentation string
}

func newStubRoot(root Root) *stubRoot {
	stubRoot := &stubRoot{
		EntryBase: NewEntry(Name(root)),
	}
	stubRoot.DisableDefaultCaching()
	schema := root.Schema()
	if schema != nil {
		stubRoot.pluginDocumentation = schema.Description
	}
	return stubRoot
}

func (r *stubRoot) Init(map[string]interface{}) error {
	return nil
}

func (r *stubRoot) Schema() *EntrySchema {
	return NewEntrySchema(r, CName(r)).
		SetDescription(r.pluginDocumentation).
		IsSingleton()
}

func (r *stubRoot) ChildSchemas() []*EntrySchema {
	return []*EntrySchema{}
}

func (r *stubRoot) List(context.Context) ([]Entry, error) {
	return []Entry{}, nil
}

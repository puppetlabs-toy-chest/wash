package apitypes

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/getlantern/deepcopy"
	"github.com/puppetlabs/wash/plugin"
)

// SignalSchema represents a signal's schema
type SignalSchema = plugin.SignalSchema

// EntrySchema describes an entry's schema, which is what's returned by
// the /fs/schema endpoint.
//
// swagger:response
type EntrySchema struct {
	plugin.EntrySchema
	path     string
	typeID   string
	children []*EntrySchema
	// graph is an ordered map of `<TypeID>` => `<EntrySchema>`. We store it to make
	// MarshalJSON's implementation easier.
	//
	// NOTE: The reason we don't synchronize children with graph is b/c entry
	// schemas are immutable. Clients that want to mess with the state (e.g. like
	// `wash find`) can do so at their own peril.
	graph *linkedhashmap.Map
}

// MarshalJSON marshals the entry's schema to JSON. It takes
// a value receiver so that the entry schema's still marshalled
// when it's referenced as an interface{} object. See
// https://stackoverflow.com/a/21394657 for more details.
func (s EntrySchema) MarshalJSON() ([]byte, error) {
	if s.graph != nil {
		return s.graph.ToJSON()
	}
	return json.Marshal(s.EntrySchema)
}

// UnmarshalJSON unmarshals the entry's schema from JSON. EntrySchemas
// are unmarshalled into their stree representation. Specifically, given
// stree output like
//
// docker
// ├── containers
// │   └── [container]
// │       ├── log
// │       ├── metadata.json
// │       └── fs
// │           ├── [dir]
// │           │   ├── [dir]
// │           │   └── [file]
// │           └── [file]
// └── volumes
//     └── [volume]
//         ├── [dir]
//         │   ├── [dir]
//         │   └── [file]
//         └── [file]
//
// The unmarshalled graph will have nodes "containers", "container", "log",
// "metadata.json", "fs", "dir", "file", "file" to mirror the first half
// of the tree (the second "file" node represents "fs"' "file" child while
// the first "file" node represents "dir"'s "file" child). The second half
// of the tree is captured by nodes "volumes", "volume", "dir", "file", "file"
// (here, the second "file" node represents "volume"'s "file" child while
// the first "file" node represents "dir"'s "file" child). The stree's root
// is the "docker" node.
//
// Unmarshalling the graph this way makes things symmetric with what a user
// sees via stree. It also simplifies the implementation of some `wash find`
// primaries.
//
// NOTE: One way to think about this representation is as follows. Consider a
// node A. Let R-B-A and R-C-D-A be two possible paths from the stree root R to
// A. Then the unmarshalled graph will have two different "A" nodes -- one for
// the path "R-B-A", and another for the path "R-C-D-A". Thus, the unmarshalled
// graph represents all possible paths from the stree root R (the starting entry)
// to its descendants (the other nodes). This is the precise definition of an
// entry's stree.
//
// NOTE: The type ID uniquely identifies a specific class of entries. The path
// identifies a specific kind of entries. For example in the above stree output,
// docker/containers/container/fs/dir and docker/volumes/volume/dir both share a
// common "volumeDir" class. However, the former represents entries that are
// directories in a Docker container's enumerated filesystem while the latter
// represents entries that are directories inside a Docker volume.
//
// NOTE: The above "path" => "kind" analogy is not always correct. For example,
// "docker/containers/container/fs/file" and "docker/containers/container/fs/dir/file"
// represent the same kind of entry (a file inside a Docker container). However,
// the analogy is good enough for most cases.
func (s *EntrySchema) UnmarshalJSON(data []byte) error {
	rawGraph := linkedhashmap.New()
	if err := rawGraph.FromJSON(data); err != nil {
		return err
	}
	if rawGraph.Size() <= 0 {
		return fmt.Errorf("expected a non-empty JSON object but got %v instead", string(data))
	}
	graph := linkedhashmap.New()

	// Convert each node in the rawGraph to a plugin.EntrySchema
	// object. This is also where we validate the data.
	var err error
	rawGraph.Each(func(key interface{}, value interface{}) {
		if err != nil {
			return
		}
		var schema plugin.EntrySchema
		err = deepcopy.Copy(&schema, value.(map[string]interface{}))
		if err != nil {
			return
		}
		if len(schema.Label) <= 0 {
			err = fmt.Errorf("label for %v was not provided", key.(string))
			return
		}
		graph.Put(key, schema)
	})
	if err != nil {
		return err
	}

	// Now fill-in the stree.
	var fillStree func(string, string, map[string]*EntrySchema) *EntrySchema
	fillStree = func(path string, typeID string, visited map[string]*EntrySchema) *EntrySchema {
		if node, ok := visited[typeID]; ok {
			return node
		}
		schema, _ := graph.Get(typeID)
		node := &EntrySchema{
			EntrySchema: schema.(plugin.EntrySchema),
			typeID:      typeID,
			path:        path,
		}
		if len(path) <= 0 {
			// This is the root
			node.path = node.Label()
		} else {
			// This is some intermediate node
			node.path = path + "/" + node.Label()
		}
		visited[typeID] = node
		for _, childTypeID := range node.EntrySchema.Children {
			node.children = append(node.children, fillStree(node.path, childTypeID, visited))
		}
		delete(visited, typeID)
		return node
	}
	it := graph.Iterator()
	it.First()
	root := fillStree("", it.Key().(string), make(map[string]*EntrySchema))
	(*s) = (*root)
	s.graph = graph

	return nil
}

// Path returns the unique path to this specific entry's schema
// (relative to the stree root). The path consists of
//    <root_label>/<parent1_label>/.../<label>
func (s *EntrySchema) Path() string {
	return s.path
}

// SetPath sets the entry's schema path. This should only be called
// by the tests.
func (s *EntrySchema) SetPath(path string) *EntrySchema {
	s.path = path
	return s
}

// TypeID returns the entry's type ID.
func (s *EntrySchema) TypeID() string {
	return s.typeID
}

// SetTypeID sets the entry's type ID. This should only be called
// by the tests.
func (s *EntrySchema) SetTypeID(typeID string) *EntrySchema {
	s.typeID = typeID
	return s
}

// Label returns the entry's label
func (s *EntrySchema) Label() string {
	return s.EntrySchema.Label
}

// Description returns the entry's description
func (s *EntrySchema) Description() string {
	return s.EntrySchema.Description
}

// SetDescription sets the entry's description. This should only be called
// by the tests.
func (s *EntrySchema) SetDescription(description string) *EntrySchema {
	s.EntrySchema.Description = description
	return s
}

// Signals returns the entry's supported signals
func (s *EntrySchema) Signals() []SignalSchema {
	return s.EntrySchema.Signals
}

// SetSignals sets the entry's supported signals. This should only be called
// by the tests.
func (s *EntrySchema) SetSignals(signals []SignalSchema) *EntrySchema {
	s.EntrySchema.Signals = signals
	return s
}

// Singleton returns true if the entry's a singleton, false otherwise.
func (s *EntrySchema) Singleton() bool {
	return s.EntrySchema.Singleton
}

// Actions returns the entry's supported actions
func (s *EntrySchema) Actions() []string {
	return s.EntrySchema.Actions
}

// SetActions sets the entry's supported actions. This should only be called
// by the tests.
func (s *EntrySchema) SetActions(actions []string) *EntrySchema {
	s.EntrySchema.Actions = actions
	return s
}

// MetaAttributeSchema returns the entry's meta attribute
// schema
func (s *EntrySchema) MetaAttributeSchema() *plugin.JSONSchema {
	return s.EntrySchema.MetaAttributeSchema
}

// SetMetaAttributeSchema sets the entry's meta attribute
// schema to s. This should only be called by the tests
func (s *EntrySchema) SetMetaAttributeSchema(schema *plugin.JSONSchema) {
	s.EntrySchema.MetaAttributeSchema = schema
}

// MetadataSchema returns the entry's metadata schema
func (s *EntrySchema) MetadataSchema() *plugin.JSONSchema {
	return s.EntrySchema.MetadataSchema
}

// SetMetadataSchema sets the entry's metadata schema to s. This should
// only be called by the tests.
func (s *EntrySchema) SetMetadataSchema(schema *plugin.JSONSchema) {
	s.EntrySchema.MetadataSchema = schema
}

// Children returns the entry's child schemas
func (s *EntrySchema) Children() []*EntrySchema {
	return s.children
}

// GetChild returns the child schema corresponding to typeID
func (s *EntrySchema) GetChild(typeID string) *EntrySchema {
	for _, child := range s.Children() {
		if child.TypeID() == typeID {
			return child
		}
	}
	return nil
}

// SetChildren sets the entry's children. This should only be called by
// the tests.
func (s *EntrySchema) SetChildren(children []*EntrySchema) *EntrySchema {
	s.children = children
	return s
}

// ToMap returns a map of <path> => <childPaths...>. It is useful for testing.
func (s *EntrySchema) ToMap() map[string][]string {
	mp := make(map[string][]string)
	var visit func(s *EntrySchema)
	visit = func(s *EntrySchema) {
		if _, ok := mp[s.Path()]; ok {
			return
		}
		mp[s.Path()] = []string{}
		for _, child := range s.Children() {
			mp[s.Path()] = append(mp[s.Path()], child.Path())
			visit(child)
		}
		sort.Strings(mp[s.Path()])
	}
	visit(s)
	return mp
}

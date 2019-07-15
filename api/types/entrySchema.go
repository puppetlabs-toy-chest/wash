package apitypes

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/getlantern/deepcopy"
	"github.com/puppetlabs/wash/plugin"
)

// EntrySchema describes an entry's schema, which
// is what's returned by the /fs/schema endpoint.
//
// swagger:response
type EntrySchema struct {
	plugin.EntrySchema
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

// UnmarshalJSON unmarshals the entry's schema from JSON.
func (s *EntrySchema) UnmarshalJSON(data []byte) error {
	rawGraph := linkedhashmap.New()
	if err := rawGraph.FromJSON(data); err != nil {
		return err
	}
	if rawGraph.Size() <= 0 {
		return fmt.Errorf("expected a non-empty JSON object but got %v instead", string(data))
	}
	s.graph = linkedhashmap.New()

	// Convert each node in the rawGraph to an *EntrySchema
	// object
	var err error
	firstElem := true
	rawGraph.Each(func(key interface{}, value interface{}) {
		if err != nil {
			return
		}
		var node *EntrySchema
		if firstElem {
			node = s
			firstElem = false
		} else {
			node = &EntrySchema{
				graph: s.graph,
			}
		}
		node.EntrySchema.TypeID = key.(string)
		err = deepcopy.Copy(&node.EntrySchema, value.(map[string]interface{}))
		if err != nil {
			return
		}
		s.graph.Put(key, node)
	})
	if err != nil {
		return err
	}

	// Fill-in the children map
	s.graph.Each(func(key interface{}, value interface{}) {
		schema := value.(*EntrySchema)
		for _, childTypeID := range schema.EntrySchema.Children {
			rawChild, _ := s.graph.Get(childTypeID)
			schema.children = append(schema.children, rawChild.(*EntrySchema))
		}
	})

	return nil
}

// TypeID returns the entry's type ID. This should be unique.
func (s *EntrySchema) TypeID() string {
	return s.EntrySchema.TypeID
}

// SetTypeID sets the entry's type ID. This should only be called
// by the tests.
func (s *EntrySchema) SetTypeID(typeID string) *EntrySchema {
	s.EntrySchema.TypeID = typeID
	return s
}

// Label returns the entry's label
func (s *EntrySchema) Label() string {
	return s.EntrySchema.Label
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

// ToMap returns a map of <typeID> => <childTypeIDs...> (i.e. its pre-serialized
// graph representation). It is useful for testing.
func (s *EntrySchema) ToMap() map[string][]string {
	mp := make(map[string][]string)
	var visit func(s *EntrySchema)
	visit = func(s *EntrySchema) {
		if _, ok := mp[s.TypeID()]; ok {
			return
		}
		mp[s.TypeID()] = []string{}
		for _, child := range s.Children() {
			mp[s.TypeID()] = append(mp[s.TypeID()], child.TypeID())
			visit(child)
		}
		sort.Strings(mp[s.TypeID()])
	}
	visit(s)
	return mp
}

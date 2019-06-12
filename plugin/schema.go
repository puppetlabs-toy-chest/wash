package plugin

import (
	"fmt"
	"reflect"
	"strings"
)

// EntrySchema represents an entry's schema.
type EntrySchema struct {
	Type      string        `json:"type"`
	Label     string        `json:"label"`
	Singleton bool          `json:"singleton"`
	Actions   []string      `json:"actions"`
	Children  []EntrySchema `json:"children"`
	entry     Entry
}

// ChildSchemas is a helper that's used to implement Parent#ChildSchemas.
//
// NOTE: "Entry" should be "EntryBase". The reason it isn't is because
// "EntryBase" doesn't implement Parent, so there is no way for Wash to
// get a Parent child's child schemas. We could move ChildSchemas over to
// EntryBase, but doing so removes the existing compile-time check on whether
// a Parent provided their child schemas.
func ChildSchemas(childBases ...Entry) []EntrySchema {
	var schemas []EntrySchema
	for _, childBase := range childBases {
		schemas = append(schemas, schema(childBase, false))
	}
	return schemas
}

// Schema returns the entry's schema. Plugin authors should use plugin.ChildSchemas
// when implementing Parent#ChildSchemas. Using Schema to do this can cause infinite
// recursion if e's children have the same type as e, which can happen if e's e.g.
// a volume directory.
func Schema(e Entry) EntrySchema {
	return schema(e, true)
}

// Common helper for Schema and ChildSchema
func schema(e Entry, includeChildren bool) EntrySchema {
	// TODO: Handle external plugin schemas
	switch e.(type) {
	case *externalPluginRoot:
		return EntrySchema{
			Type: "external-plugin-root",
		}
	case *externalPluginEntry:
		return EntrySchema{
			Type: "external-plugin-entry",
		}
	}

	s := EntrySchema{
		Type:      strings.TrimPrefix(reflect.TypeOf(e).String(), "*"),
		Label:     e.entryBase().label,
		Singleton: e.entryBase().isSingleton,
		Actions:   SupportedActionsOf(e),
		entry:     e,
	}
	if len(s.Label) == 0 {
		msg := fmt.Sprintf("Schema for type %v has an empty label. Use EntryBase#SetLabel to set the label.", s.Type)
		panic(msg)
	}
	if includeChildren {
		s.fillChildren(make(map[string]bool))
	}
	return s
}

func (s *EntrySchema) fillChildren(visited map[string]bool) {
	if !ListAction().IsSupportedOn(s.entry) {
		return
	}
	if visited[s.Type] {
		// This means that s' children can have s' type, which is
		// true if s is e.g. a volume directory.
		return
	}
	s.Children = s.entry.(Parent).ChildSchemas()
	visited[s.Type] = true
	for i, child := range s.Children {
		child.fillChildren(visited)
		// Need to re-assign because child is not a pointer,
		// so s.Children[i] won't be updated.
		s.Children[i] = child
	}
	// Delete "s" from visited so that siblings or ancestors that
	// also use "s" won't be affected.
	delete(visited, s.Type)
}

package plugin

import (
	"fmt"
	"reflect"
	"strings"
)

// EntrySchema represents an entry's schema.
type EntrySchema struct {
	TypeID              string         `json:"type_id"`
	Label               string         `json:"label"`
	Singleton           bool           `json:"singleton"`
	Actions             []string       `json:"actions"`
	MetaAttributeSchema JSONSchema     `json:"meta_attribute_schema"`
	MetadataSchema      JSONSchema     `json:"metadata_schema"`
	Children            []*EntrySchema `json:"children"`
	entry               Entry
}

// ChildSchemas is a helper that's used to implement Parent#ChildSchemas.
//
// NOTE: "Entry" should be "EntryBase". The reason it isn't is because
// "EntryBase" doesn't implement Parent, so there is no way for Wash to
// get a Parent child's child schemas. We could move ChildSchemas over to
// EntryBase, but doing so removes the existing compile-time check on whether
// a Parent provided their child schemas.
func ChildSchemas(childBases ...Entry) []*EntrySchema {
	var schemas []*EntrySchema
	for _, childBase := range childBases {
		schemas = append(schemas, schema(childBase, false))
	}
	return schemas
}

// TypeID returns the entry's type ID
func TypeID(e Entry) string {
	// TODO: Handle external plugin type IDs.
	return strings.TrimPrefix(reflect.TypeOf(e).String(), "*")
}

// Schema returns the entry's schema. Plugin authors should use plugin.ChildSchemas
// when implementing Parent#ChildSchemas. Using Schema to do this can cause infinite
// recursion if e's children have the same type as e, which can happen if e's e.g.
// a volume directory.
func Schema(e Entry) *EntrySchema {
	return schema(e, true)
}

// Common helper for Schema and ChildSchema
func schema(e Entry, includeChildren bool) *EntrySchema {
	// TODO: Handle external plugin schemas
	switch e.(type) {
	case *externalPluginRoot:
		return nil
	case *externalPluginEntry:
		return nil
	}

	s := &EntrySchema{
		TypeID:              TypeID(e),
		Label:               e.entryBase().label,
		Singleton:           e.entryBase().isSingleton,
		Actions:             SupportedActionsOf(e),
		MetaAttributeSchema: e.entryBase().metaAttributeSchema,
		MetadataSchema:      e.entryBase().metadataSchema,
		entry:               e,
	}
	if len(s.Label) == 0 {
		msg := fmt.Sprintf("Schema for type %v has an empty label. Use EntryBase#SetLabel to set the label.", s.TypeID)
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
	if visited[s.TypeID] {
		// This means that s' children can have s' type, which is
		// true if s is e.g. a volume directory.
		return
	}
	children := s.entry.(Parent).ChildSchemas()
	visited[s.TypeID] = true
	for _, child := range children {
		if child == nil {
			continue
		}
		s.Children = append(s.Children, child)
		child.fillChildren(visited)
	}
	// Delete "s" from visited so that siblings or ancestors that
	// also use "s" won't be affected.
	delete(visited, s.TypeID)
}

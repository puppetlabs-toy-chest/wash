package plugin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ekinanp/jsonschema"
)

type entrySchema struct {
	TypeID              string         `json:"type_id"`
	Label               string         `json:"label"`
	Singleton           bool           `json:"singleton"`
	Actions             []string       `json:"actions"`
	MetaAttributeSchema *JSONSchema    `json:"meta_attribute_schema"`
	MetadataSchema      *JSONSchema    `json:"metadata_schema"`
	Children            []*EntrySchema `json:"children"`
	entry               Entry
}

// EntrySchema represents an entry's schema. Use plugin.NewEntrySchema
// to create instances of these objects.
//
// EntrySchema's a useful way to document your plugin's hierarchy. Users
// can view your hierarchy via the stree command. For example, if you
// invoke `stree docker` in a Wash shell (try it!), you should see something
// like
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
// 		└── [file]
//
// (Your output may differ depending on the state of the Wash project, but it
// should be similarly structured).
//
// Every node must have a label. The "[]" are printed for non-singleton nodes;
// they imply multiple instances of this thing. For example, "[container]" means
// that there will be multiple "container" instances under the "containers" directory
// ("container" is the label that was passed into NewEntrySchema). Similarly, "containers"
// means that there will be only one "containers" directory (i.e. that "containers" is a
// singleton). You can use EntrySchema#IsSingleton() to mark your entry as a singleton.
//
// TODO: Talk about how metadata schema's used to optimize `wash find` once that
// is added.
type EntrySchema struct {
	// This pattern's a nice way of making JSON marshalling/unmarshalling
	// easy without having to export these fields via the godocs. The latter
	// is good because plugin authors should use the builders when setting them
	// (so that we maintain a consistent API for e.g. metadata schemas).
	//
	// This pattern was obtained from https://stackoverflow.com/a/11129474
	entrySchema
	metaAttributeSchemaObj interface{}
	metadataSchemaObj      interface{}
	partialSchema          bool
}

// NewEntrySchema returns a new EntrySchema object with the specified label.
//
// NOTE: If your entry's a singleton, then the label should match the entry's
// name, i.e. the name that's passed into plugin.NewEntry.
func NewEntrySchema(e Entry, label string) *EntrySchema {
	s := &EntrySchema{
		entrySchema: entrySchema{
			TypeID:  strings.TrimPrefix(reflect.TypeOf(e).String(), "*"),
			Actions: SupportedActionsOf(e),
			// Store the entry so that when marshalling, we can enumerate
			// its child schemas.
			entry: e,
		},
		// The meta attribute's empty by default
		metaAttributeSchemaObj: struct{}{},
		partialSchema:          true,
	}
	s.SetLabel(label)
	return s
}

// MarshalJSON marshals the entry's schema to JSON. It takes
// a value receiver so that the entry schema's still marshalled
// when it's referenced as an interface{} object. See
// https://stackoverflow.com/a/21394657 for more details.
func (s EntrySchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.entrySchema)
}

// UnmarshalJSON unmarshals the entry's schema from JSON.
func (s *EntrySchema) UnmarshalJSON(data []byte) error {
	var es entrySchema
	if err := json.Unmarshal(data, &es); err != nil {
		return err
	}
	s.entrySchema = es
	return nil
}

// TypeID returns the entry's type ID. This should be unique.
func (s *EntrySchema) TypeID() string {
	return s.entrySchema.TypeID
}

// SetTypeID sets the entry's type ID. Only the tests
// can call this method.
func (s *EntrySchema) SetTypeID(typeID string) *EntrySchema {
	if notRunningTests() {
		panic("s.SetTypeID can only be called by the tests")
	}
	s.entrySchema.TypeID = typeID
	return s
}

// Label returns the entry's label
func (s *EntrySchema) Label() string {
	return s.entrySchema.Label
}

/*
SetLabel overrides the entry's label. You should only override the
label if you are extending a helper type or are using that helper
type as part of your entry's child schema. See docker.Container#ChildSchema
for an example of how this is used.
*/
func (s *EntrySchema) SetLabel(label string) *EntrySchema {
	if len(label) == 0 {
		panic("s.SetLabel called with an empty label")
	}
	s.entrySchema.Label = label
	return s
}

// Singleton returns true if the entry's a singleton, false otherwise.
func (s *EntrySchema) Singleton() bool {
	return s.entrySchema.Singleton
}

// IsSingleton marks the entry as a singleton entry.
func (s *EntrySchema) IsSingleton() *EntrySchema {
	s.entrySchema.Singleton = true
	return s
}

// Actions returns the entry's supported actions
func (s *EntrySchema) Actions() []string {
	return s.entrySchema.Actions
}

// SetActions sets the entry's supported actions. Only
// the tests can call this method.
func (s *EntrySchema) SetActions(actions []string) *EntrySchema {
	if notRunningTests() {
		panic("s.SetActions can only be called by the tests")
	}
	s.entrySchema.Actions = actions
	return s
}

// MetaAttributeSchema returns the entry's meta attribute
// schema
func (s *EntrySchema) MetaAttributeSchema() *JSONSchema {
	return s.entrySchema.MetaAttributeSchema
}

// SetMetaAttributeSchema sets the meta attribute's schema. obj is an empty struct
// that will be marshalled into a JSON schema. SetMetaSchema will panic
// if obj is not a struct.
func (s *EntrySchema) SetMetaAttributeSchema(obj interface{}) *EntrySchema {
	// We need to know if s.entry has any wrapped types in order to correctly
	// compute the schema. However that information is known when s.fill() is
	// called. Thus, we'll set the schema object to obj so s.fill() can properly
	// calculate the schema.
	s.metaAttributeSchemaObj = obj
	return s
}

// SetTestMetaAttributeSchema sets the entry's meta attribute
// schema to s. Only the tests can call this method.
func (s *EntrySchema) SetTestMetaAttributeSchema(schema *JSONSchema) {
	if notRunningTests() {
		panic("s.SetTestMetaAttributeSchema can only be called by the tests")
	}
	s.entrySchema.MetaAttributeSchema = schema
}

// MetadataSchema returns the entry's metadata schema
func (s *EntrySchema) MetadataSchema() *JSONSchema {
	return s.entrySchema.MetadataSchema
}

// SetMetadataSchema sets Entry#Metadata's schema. obj is an empty struct that will be
// marshalled into a JSON schema. SetMetadataSchema will panic if obj is not a struct.
//
// NOTE: Only use SetMetadataSchema if you're overriding Entry#Metadata. Otherwise, use
// SetMetaAttributeSchema.
func (s *EntrySchema) SetMetadataSchema(obj interface{}) *EntrySchema {
	// See the comments in SetMetaAttributeSchema to understand why this line's necessary
	s.metadataSchemaObj = obj
	return s
}

// SetTestMetadataSchema sets the entry's metadata schema to s. Only the tests can
// call this method.
func (s *EntrySchema) SetTestMetadataSchema(schema *JSONSchema) {
	if notRunningTests() {
		panic("s.SetTestMetadataSchema can only be called by the tests")
	}
	s.entrySchema.MetadataSchema = schema
}

// Children returns the entry's child schemas
func (s *EntrySchema) Children() []*EntrySchema {
	return s.entrySchema.Children
}

// SetChildren sets the entry's children. Only the tests
// can call this method.
func (s *EntrySchema) SetChildren(children []*EntrySchema) *EntrySchema {
	if notRunningTests() {
		panic("s.SetChildren can only be called by the tests")
	}
	s.entrySchema.Children = children
	return s
}

// Fill fills s' children, its meta attribute schema, and its metadata
// schema. This is needed by the API. Plugin authors should ignore this.
func (s *EntrySchema) Fill() {
	if s.partialSchema {
		s.fill(make(map[string]bool))
		s.partialSchema = false
	}
}

func (s *EntrySchema) fill(visited map[string]bool) {
	// Fill-in the meta attribute + metadata schemas
	//
	// TODO: This causes duplicate metadata schemas to be returned
	// for nodes that are part of a cycle. We will fix that once we
	// clean up /fs/schema's response to adopt a representation that's
	// more idiomatic with graph serialization. For now, we need this
	// hack for metadata schemas to work correctly in `wash find`.
	var err error
	if s.metaAttributeSchemaObj != nil {
		s.entrySchema.MetaAttributeSchema, err = s.schemaOf(s.metaAttributeSchemaObj)
		if err != nil {
			s.fillPanicf("bad value passed into SetMetaAttributeSchema: %v", err)
		}
	}
	if s.metadataSchemaObj != nil {
		s.entrySchema.MetadataSchema, err = s.schemaOf(s.metadataSchemaObj)
		if err != nil {
			s.fillPanicf("bad value passed into SetMetadataSchema: %v", err)
		}
	}

	if visited[s.TypeID()] {
		// This means that s' children can have s' type, which is
		// true if s is e.g. a volume directory.
		return
	}
	visited[s.TypeID()] = true

	// Fill-in the children
	if !ListAction().IsSupportedOn(s.entry) {
		return
	}
	// "sParent" is read as "s.parent"
	sParent := s.entry.(Parent)
	children := sParent.ChildSchemas()
	for _, child := range children {
		if child == nil {
			s.fillPanicf("found a nil child schema")
		}
		s.entrySchema.Children = append(s.entrySchema.Children, child)
		passAlongWrappedTypes(sParent, child.entry)
		child.fill(visited)
	}
	// Delete "s" from visited so that siblings or ancestors that
	// also use "s" won't be affected.
	delete(visited, s.TypeID())
}

// This helper's used by CachedList + EntrySchema#fill(). The reason for
// the helper is because /fs/schema uses repeated calls to CachedList when
// fetching the entry, so we need to pass-along the wrapped types when
// searching for it. However, Parent#ChildSchemas uses empty Entry objects
// that do not go through CachedList (by definition). Thus, the entry found
// by /fs/schema needs to pass its wrapped types along to the children to
// determine their metadata schemas. This is done in s.fill().
func passAlongWrappedTypes(p Parent, child Entry) {
	var wrappedTypes SchemaMap
	if root, ok := child.(HasWrappedTypes); ok {
		wrappedTypes = root.WrappedTypes()
	} else {
		wrappedTypes = p.wrappedTypes()
	}
	child.setWrappedTypes(wrappedTypes)
}

// Helper that wraps the common code shared by the SetMeta*Schema methods
func (s *EntrySchema) schemaOf(obj interface{}) (*JSONSchema, error) {
	typeMappings := make(map[reflect.Type]*jsonschema.Type)
	for t, s := range s.entry.wrappedTypes() {
		typeMappings[reflect.TypeOf(t)] = s.Type
	}
	r := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		// Setting this option ensures that the schema's root is obj's
		// schema instead of a reference to a definition containing obj's
		// schema. This way, we can validate that "obj" is a JSON object's
		// schema. Otherwise, the check below will always fail.
		ExpandedStruct: true,
		TypeMappings:   typeMappings,
	}
	schema := r.Reflect(obj)
	if schema.Type.Type != "object" {
		return nil, fmt.Errorf("expected a JSON object but got %v", schema.Type.Type)
	}
	return schema, nil
}

// Helper for s.fill(). We make it a separate method to avoid re-creating
// closures for each recursive s.fill() call.
func (s *EntrySchema) fillPanicf(format string, a ...interface{}) {
	formatStr := fmt.Sprintf("s.fill (%v): %v", s.TypeID(), format)
	msg := fmt.Sprintf(formatStr, a...)
	panic(msg)
}

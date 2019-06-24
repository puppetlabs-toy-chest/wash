package plugin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type entrySchema struct {
	TypeID    string         `json:"type_id"`
	Label     string         `json:"label"`
	Singleton bool           `json:"singleton"`
	Actions   []string       `json:"actions"`
	Children  []*EntrySchema `json:"children"`
	entry     Entry
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

// FillChildren fills s' children.
func (s *EntrySchema) FillChildren() {
	s.fillChildren(make(map[string]bool))
}

func (s *EntrySchema) fillChildren(visited map[string]bool) {
	if s.entrySchema.Children != nil {
		return
	}
	if !ListAction().IsSupportedOn(s.entry) {
		return
	}
	if visited[s.TypeID()] {
		// This means that s' children can have s' type, which is
		// true if s is e.g. a volume directory.
		return
	}
	children := s.entry.(Parent).ChildSchemas()
	visited[s.TypeID()] = true
	for _, child := range children {
		if child == nil {
			msg := fmt.Sprintf("s.fillChildren: found a nil child schema for %v", s.TypeID())
			panic(msg)
		}
		s.entrySchema.Children = append(s.entrySchema.Children, child)
		child.fillChildren(visited)
	}
	// Delete "s" from visited so that siblings or ancestors that
	// also use "s" won't be affected.
	delete(visited, s.TypeID())
}

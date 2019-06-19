package plugin

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"time"

	"github.com/ekinanp/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type defaultOpCode int8

const (
	// ListOp represents Parent#List
	ListOp defaultOpCode = iota
	// OpenOp represents Readable#Open
	OpenOp
	// MetadataOp represents Entry#Metadata
	MetadataOp
)

var defaultOpCodeToNameMap = [3]string{"List", "Open", "Metadata"}

// JSONSchema represents a JSON schema
type JSONSchema *jsonschema.Schema

/*
EntryBase implements Entry, making it easy to create new entries.
You should use plugin.NewEntryBaseBase to create new EntryBase objects.

Each of the setters supports the builder pattern, which enables you
to do something like

	e := plugin.NewEntryBaseBase()
	e.
		DisableCachingFor(plugin.ListOp).
		Attributes().
		SetCtime(ctime).
		SetMtime(mtime).
		SetMeta(meta)
*/
type EntryBase struct {
	entryName       string
	attr            EntryAttributes
	slashReplacerCh rune
	// washID represents the entry's wash ID. It is set in CachedList.
	washID              string
	ttl                 [3]time.Duration
	label               string
	isSingleton         bool
	metaAttributeSchema JSONSchema
	metadataSchema      JSONSchema
}

var defaultMetaAttributeSchema = jsonschema.Reflect(struct{}{})

// NewEntryBase creates a new EntryBase object
func NewEntryBase() EntryBase {
	e := EntryBase{
		slashReplacerCh: '#',
	}
	for op := range e.ttl {
		e.SetTTLOf(defaultOpCode(op), 15*time.Second)
	}
	e.metaAttributeSchema = defaultMetaAttributeSchema
	return e
}

// ENTRY INTERFACE

// Metadata returns the entry's meta attribute (see plugin.EntryAttributes).
// Override this if e has additional metadata.
func (e *EntryBase) Metadata(ctx context.Context) (JSONObject, error) {
	// Disable Metadata's caching in case the plugin author forgot to do this
	e.DisableCachingFor(MetadataOp)

	attr := e.attributes()
	return attr.Meta(), nil
}

func (e *EntryBase) name() string {
	return e.entryName
}

func (e *EntryBase) attributes() EntryAttributes {
	return e.attr
}

func (e *EntryBase) slashReplacer() rune {
	return e.slashReplacerCh
}

func (e *EntryBase) id() string {
	return e.washID
}

func (e *EntryBase) setID(id string) {
	e.washID = id
}

func (e *EntryBase) getTTLOf(op defaultOpCode) time.Duration {
	return e.ttl[op]
}

func (e *EntryBase) entryBase() *EntryBase {
	return e
}

// OTHER METHODS USED TO FACILITATE PLUGIN DEVELOPMENT
// AND TESTING

// Name returns the entry's name as it was passed into
// e.SetName. You should use e.Name() when making the
// appropriate API calls within your plugin.
func (e *EntryBase) Name() string {
	return e.name()
}

// SetName sets the entry's name.
func (e *EntryBase) SetName(name string) *EntryBase {
	if name == "" {
		panic("e.SetName: received an empty name")
	}
	e.entryName = name
	return e
}

// Attributes returns a pointer to the entry's attributes. Use it
// to individually set the entry's attributes
func (e *EntryBase) Attributes() *EntryAttributes {
	return &e.attr
}

// SetAttributes sets the entry's attributes. Use it to set
// the entry's attributes in a single operation, which is useful
// when you've already pre-computed them.
func (e *EntryBase) SetAttributes(attr EntryAttributes) *EntryBase {
	e.attr = attr
	return e
}

/*
SetSlashReplacer overrides the default '/' replacer '#' to char.
The '/' replacer is used when determining the entry's cname. See
plugin.CName for more details.
*/
func (e *EntryBase) SetSlashReplacer(char rune) *EntryBase {
	if char == '/' {
		panic("e.SetSlashReplacer called with '/'")
	}

	e.slashReplacerCh = char
	return e
}

/*
SetLabel sets the entry's label, which is a shortened version
of the class/struct name. It is useful when documenting your
plugin's hierarchy.

NOTE: If your entry's a singleton, then Wash will default to using the
entry's cname as its label.

TODO: Give an example of why this is important once the stree command's
implemented.
*/
func (e *EntryBase) SetLabel(label string) *EntryBase {
	if len(label) == 0 {
		panic("e.SetLabel called with an empty label")
	}
	e.label = label
	return e
}

/*
IsSingleton indicates that the given entry's a singleton, meaning there
will only ever be one instance of the entry. It is useful when documenting
your plugin's hierarchy.

NOTE: If the entry's short type was not set, then IsSingleton sets it to
the entry's cname.

TODO: Give an example of why this is important once the stree command's
implemented.
*/
func (e *EntryBase) IsSingleton() *EntryBase {
	e.isSingleton = true
	if len(e.label) == 0 {
		e.SetLabel(CName(e))
	}
	return e
}

// SetMetaAttributeSchema sets the meta attribute's schema. obj is an empty struct
// that will be marshalled into a JSON schema. SetMetaSchema will panic
// if obj is not a struct.
func (e *EntryBase) SetMetaAttributeSchema(obj interface{}) *EntryBase {
	s, err := schemaOf(obj)
	if err != nil {
		panic(fmt.Sprintf("e.SetMetaAttributeSchema: %v", err))
	}
	e.metaAttributeSchema = s
	return e
}

// SetMetadataSchema sets e#Metadata's schema. obj is an empty struct
// that will be marshalled into a JSON schema. SetMetadataSchema will
// panic if obj is not a struct.
//
// NOTE: Only use SetMetadataSchema if you're overriding e#Metadata.
// Otherwise, use SetMetaAttributeSchema.
func (e *EntryBase) SetMetadataSchema(obj interface{}) *EntryBase {
	s, err := schemaOf(obj)
	if err != nil {
		panic(fmt.Sprintf("e.SetMetadataSchema: %v", err))
	}
	e.metadataSchema = s
	return e
}

// SetTTLOf sets the specified op's TTL
func (e *EntryBase) SetTTLOf(op defaultOpCode, ttl time.Duration) *EntryBase {
	e.ttl[op] = ttl
	return e
}

// DisableCachingFor disables caching for the specified op
func (e *EntryBase) DisableCachingFor(op defaultOpCode) *EntryBase {
	e.SetTTLOf(op, -1)
	return e
}

// DisableDefaultCaching disables the default caching
// for List, Open and Metadata.
func (e *EntryBase) DisableDefaultCaching() *EntryBase {
	for op := range e.ttl {
		e.DisableCachingFor(defaultOpCode(op))
	}
	return e
}

// SetTestID sets the entry's cache ID for testing.
// It can only be called by the tests.
func (e *EntryBase) SetTestID(id string) {
	if notRunningTests() {
		panic("SetTestID can be only be called by the tests")
	}

	e.setID(id)
}

func notRunningTests() bool {
	return flag.Lookup("test.v") == nil
}

var aliases = func() map[reflect.Type]*jsonschema.Type {
	mp := make(map[reflect.Type]*jsonschema.Type)
	// v1.Time is an alias to a time.Time object
	mp[reflect.TypeOf(v1.Time{})] = jsonschema.TimeType
	mp[reflect.TypeOf(resource.Quantity{})] = jsonschema.NumberType
	return mp
}()

// Helper that wraps the common code shared by
// the SetMeta*Schema methods
func schemaOf(obj interface{}) (JSONSchema, error) {
	r := jsonschema.Reflector{
		// Setting this option ensures that the schema's root is obj's
		// schema instead of a reference to a definition containing obj's
		// schema. This way, we can validate that "obj" is a JSON object's
		// schema. Otherwise, the check below will always fail.
		ExpandedStruct: true,
		Aliases: aliases,
	}
	s := r.Reflect(obj)
	if s.Type.Type != "object" {
		return nil, fmt.Errorf("expected a JSON object but got %v", s.Type.Type)
	}
	return s, nil
}

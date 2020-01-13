package plugin

import (
	"context"
	"flag"
	"time"

	"github.com/puppetlabs/wash/activity"
)

type defaultOpCode int8

const (
	// ListOp represents Parent#List
	ListOp defaultOpCode = iota
	// ReadOp represents Readable/BlockReadable#Read
	ReadOp
	// MetadataOp represents Entry#Metadata
	MetadataOp
)

var defaultOpCodeToNameMap = [3]string{"List", "Read", "Metadata"}

/*
EntryBase implements Entry, making it easy to create new entries.
You should use plugin.NewEntry to create new EntryBase objects.

Each of the setters supports the builder pattern, which enables you
to do something like

	e := plugin.NewEntry("foo")
	e.
		DisableCachingFor(plugin.ListOp).
		Attributes().
		SetCrtime(crtime).
		SetMtime(mtime).
		SetMeta(meta)
*/
type EntryBase struct {
	name                     string
	attributes               EntryAttributes
	specifiedPartialMetadata JSONObject
	slashReplacer            rune
	id                       string
	ttl                      [3]time.Duration
	wrappedTypes             SchemaMap
	isPrefetched             bool
	isInaccessible           bool
}

// NewEntry creates a new entry
func NewEntry(name string) EntryBase {
	if name == "" {
		panic("plugin.NewEntry: received an empty name")
	}

	e := EntryBase{
		name:          name,
		slashReplacer: '#',
	}
	for op := range e.ttl {
		e.SetTTLOf(defaultOpCode(op), 15*time.Second)
	}
	return e
}

func (e *EntryBase) partialMetadata() JSONObject {
	if e.specifiedPartialMetadata != nil {
		return e.specifiedPartialMetadata
	}
	return e.attributes.ToMap()
}

// ENTRY INTERFACE

// Metadata returns the entry's partial metadata. Override this if e has
// additional metadata.
func (e *EntryBase) Metadata(ctx context.Context) (JSONObject, error) {
	// Disable Metadata's caching in case the plugin author forgot to do this
	e.DisableCachingFor(MetadataOp)
	return e.partialMetadata(), nil
}

func (e *EntryBase) eb() *EntryBase {
	return e
}

// OTHER METHODS USED TO FACILITATE PLUGIN DEVELOPMENT
// AND TESTING

// Name returns the entry's name as it was passed into
// plugin.NewEntry. You should use e.Name() when making
// the appropriate API calls within your plugin.
func (e *EntryBase) Name() string {
	return e.name
}

// String returns a unique identifier for the entry suitable for logging and error messages.
func (e *EntryBase) String() string {
	return e.id
}

// Attributes returns a pointer to the entry's attributes. Use it
// to individually set the entry's attributes
func (e *EntryBase) Attributes() *EntryAttributes {
	return &e.attributes
}

// SetAttributes sets the entry's attributes. Use it to set
// the entry's attributes in a single operation, which is useful
// when you've already pre-computed them.
func (e *EntryBase) SetAttributes(attr EntryAttributes) *EntryBase {
	e.attributes = attr
	return e
}

// SetPartialMetadata sets the entry's partial metadata. This is typically the
// raw object that's returned by the plugin API's List endpoint, or a wrapper
// that includes the raw object + some additional information. For example, if
// the entry represents a Docker container, then obj would be a Container struct.
// If the entry represents a Docker volume, then obj would be a Volume struct.
//
// SetPartialMetadata will panic if obj does not serialize to a JSON object.
func (e *EntryBase) SetPartialMetadata(obj interface{}) *EntryBase {
	e.specifiedPartialMetadata = ToJSONObject(obj)
	return e
}

// MarkInaccessible sets the inaccessible attribute and logs a message about why the entry is
// inaccessible.
func (e *EntryBase) MarkInaccessible(ctx context.Context, err error) {
	activity.Record(ctx, "Omitting %v: %v", e.id, err)
	e.isInaccessible = true
}

// Prefetched marks the entry as a prefetched entry. A prefetched entry
// is an entry that was fetched as part of a batch operation that
// fetched multiple levels of hierarchy at once. Volume directories and
// files are good examples of prefetched entries (see the volume
// package for more details).
func (e *EntryBase) Prefetched() *EntryBase {
	e.isPrefetched = true
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

	e.slashReplacer = char
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

	e.id = id
}

func notRunningTests() bool {
	return flag.Lookup("test.v") == nil
}

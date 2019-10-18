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
	// OpenOp represents Readable#Open
	OpenOp
	// MetadataOp represents Entry#Metadata
	MetadataOp
)

var defaultOpCodeToNameMap = [3]string{"List", "Open", "Metadata"}

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
	entryName       string
	attr            EntryAttributes
	slashReplacerCh rune
	// washID represents the entry's wash ID. It is set in CachedList.
	washID          string
	ttl             [3]time.Duration
	wrappedTypesMap SchemaMap
	prefetched      bool
	inaccessible    bool
}

// NewEntry creates a new entry
func NewEntry(name string) EntryBase {
	if name == "" {
		panic("plugin.NewEntry: received an empty name")
	}

	e := EntryBase{
		entryName:       name,
		slashReplacerCh: '#',
	}
	for op := range e.ttl {
		e.SetTTLOf(defaultOpCode(op), 15*time.Second)
	}
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

func (e *EntryBase) wrappedTypes() SchemaMap {
	return e.wrappedTypesMap
}

func (e *EntryBase) setWrappedTypes(wrappedTypes SchemaMap) {
	e.wrappedTypesMap = wrappedTypes
}

func (e *EntryBase) isPrefetched() bool {
	return e.prefetched
}

// OTHER METHODS USED TO FACILITATE PLUGIN DEVELOPMENT
// AND TESTING

// Name returns the entry's name as it was passed into
// plugin.NewEntry. You should use e.Name() when making
// the appropriate API calls within your plugin.
func (e *EntryBase) Name() string {
	return e.name()
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

func (e *EntryBase) isInaccessible() bool {
	return e.inaccessible
}

// MarkInaccessible sets the inaccessible attribute and logs a message about why the entry is
// inaccessible.
func (e *EntryBase) MarkInaccessible(ctx context.Context, err error) {
	activity.Record(ctx, "Omitting %v: %v", e.id(), err)
	e.inaccessible = true
}

// Prefetched marks the entry as a prefetched entry. A prefetched entry
// is an entry that was fetched as part of a batch operation that
// fetched multiple levels of hierarchy at once. Volume directories and
// files are good examples of prefetched entries (see the volume
// package for more details).
func (e *EntryBase) Prefetched() *EntryBase {
	e.prefetched = true
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

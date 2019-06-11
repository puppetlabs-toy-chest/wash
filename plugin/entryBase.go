package plugin

import (
	"context"
	"flag"
	"time"
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
	entryName          string
	attr               EntryAttributes
	slashReplacerCh    rune
	// washID represents the entry's wash ID. It is set in CachedList.
	washID             string
	ttl                [3]time.Duration
	_shortType         string
	singleton          bool
}

// NewEntryBase creates a new EntryBase object
func NewEntryBase() EntryBase {
	e := EntryBase{
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

func (e *EntryBase) shortType() string {
	return e._shortType
}

func (e *EntryBase) isSingleton() bool {
	return e.singleton
}

func (e *EntryBase) markSingleton() {
	e.IsSingleton()
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
SetShortType sets the entry's short type, which is a shortened version
of the class/struct name. It is useful when documenting your plugin's
hierarchy.

NOTE: If your entry's a singleton, then Wash will default to using the
entry's cname as its short type

TODO: Give an example of why this is important once the stree command's
implemented.
*/
func (e *EntryBase) SetShortType(shortType string) *EntryBase {
	if len(shortType) == 0 {
		panic("e.SetShortType called with an empty shortType")
	}
	e._shortType = shortType
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
	e.singleton = true
	if len(e._shortType) == 0 {
		e.SetShortType(CName(e))
	}
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

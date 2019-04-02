package plugin

import (
	"context"
	"flag"
	"time"
)

type defaultOpCode int8

const (
	// ListOp represents Group#List
	ListOp defaultOpCode = iota
	// OpenOp represents Readable#Open
	OpenOp
	// MetadataOp represents Entry#Metadata
	MetadataOp
)

var defaultOpCodeToNameMap = [3]string{"List", "Open", "Metadata"}

// EntryBase implements Entry, making it easy to create new entries.
// You should use plugin.NewEntry to create new EntryBase objects.
type EntryBase struct {
	entryName          string
	attr               EntryAttributes
	slashReplacementCh rune
	// washID represents the entry's wash ID. It is set in CachedList.
	washID string
	ttl    [3]time.Duration
}

// NewEntry creates a new entry
func NewEntry(name string) EntryBase {
	if name == "" {
		panic("plugin.NewEntry: received an empty name")
	}

	e := EntryBase{
		entryName:          name,
		slashReplacementCh: '#',
	}
	for op := range e.ttl {
		e.SetTTLOf(defaultOpCode(op), 15*time.Second)
	}
	return e
}

// ENTRY INTERFACE

// Metadata returns the entry's meta attribute (see plugin.EntryAttributes).
// Do not override this if the entry's metadata will never change.
func (e *EntryBase) Metadata(ctx context.Context) (EntryMetadata, error) {
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

func (e *EntryBase) slashReplacementChar() rune {
	return e.slashReplacementCh
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

// OTHER METHODS USED TO FACILITATE PLUGIN DEVELOPMENT
// AND TESTING

// Name returns the entry's name as it was passed into
// plugin.NewEntry. You should use e.Name() when making
// the appropriate API calls within your plugin.
func (e *EntryBase) Name() string {
	return e.name()
}

// SetInitialAttributes sets the entry's initial attributes. Use it
// after creating the entry via a call to NewEntry.
func (e *EntryBase) SetInitialAttributes(attr EntryAttributes) {
	e.attr = attr
}

/*
SetSlashReplacementChar overrides the default '/' replacement
character of '#' to char. The '/' replacement character is used
when determining the entry's cname. See plugin.CName for more
details.
*/
func (e *EntryBase) SetSlashReplacementChar(char rune) {
	if char == '/' {
		panic("e.SetSlashReplacementChar called with '/'")
	}

	e.slashReplacementCh = char
}

// SetTTLOf sets the specified op's TTL
func (e *EntryBase) SetTTLOf(op defaultOpCode, ttl time.Duration) {
	e.ttl[op] = ttl
}

// DisableCachingFor disables caching for the specified op
func (e *EntryBase) DisableCachingFor(op defaultOpCode) {
	e.SetTTLOf(op, -1)
}

// DisableDefaultCaching disables the default caching
// for List, Open and Metadata.
func (e *EntryBase) DisableDefaultCaching() {
	for op := range e.ttl {
		e.DisableCachingFor(defaultOpCode(op))
	}
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

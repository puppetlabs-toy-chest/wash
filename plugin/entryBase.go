package plugin

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"
)

type actionOpCode int8

const (
	// List represents Group#List
	List actionOpCode = iota
	// Open represents Readable#Open
	Open
	// Metadata represents Resource#Metadata
	Metadata
)

var actionOpCodeToNameMap = [3]string{"List", "Open", "Metadata"}

// EntryBase implements Entry, making it easy to create new entries.
// You should use plugin.NewEntry to create new EntryBase objects.
type EntryBase struct {
	name string
	// id represents the entry's wash ID. It is set in CachedList.
	id   string
	attr Attributes
	ttl  [3]time.Duration
}

// newEntryBase is needed by NewEntry, NewRegistry,
// and some of the cache tests
func newEntryBase(name string) EntryBase {
	e := EntryBase{name: name}

	for op := range e.ttl {
		e.SetTTLOf(actionOpCode(op), 15*time.Second)
	}

	return e
}

// NewEntry creates a new entry
func NewEntry(name string) EntryBase {
	if name == "" {
		panic("plugin.NewEntry: received an empty name")
	}
	if strings.Contains(name, "/") {
		panic(fmt.Sprintf("plugin.NewEntry: received name %v containing a /", name))
	}

	return newEntryBase(name)
}

// ENTRY INTERFACE

// Name returns the entry's name.
func (e *EntryBase) Name() string {
	return e.name
}

// ID returns the entry's wash ID
func (e *EntryBase) ID() string {
	return e.id
}

// Attr returns the entry's attributes
func (e *EntryBase) Attr(ctx context.Context) (Attributes, error) {
	return e.attr, nil
}

func (e *EntryBase) getTTLOf(op actionOpCode) time.Duration {
	return e.ttl[op]
}

func (e *EntryBase) setID(id string) {
	e.id = id
}

// OTHER METHODS USED TO FACILITATE PLUGIN DEVELOPMENT
// AND TESTING

// SetTTLOf sets the specified op's TTL
func (e *EntryBase) SetTTLOf(op actionOpCode, ttl time.Duration) {
	e.ttl[op] = ttl
}

// TurnOffCachingFor turns off caching for the specified op
func (e *EntryBase) TurnOffCachingFor(op actionOpCode) {
	e.SetTTLOf(op, -1)
}

// TurnOffCaching turns off caching for all ops
func (e *EntryBase) TurnOffCaching() {
	for op := range e.ttl {
		e.TurnOffCachingFor(actionOpCode(op))
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

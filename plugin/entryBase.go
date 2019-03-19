package plugin

import (
	"flag"
	"time"
)

type cacheableOp int8

const (
	// List represents Group#List
	List cacheableOp = iota
	// Open represents Readable#Open
	Open
	// Metadata represents Resource#Metadata
	Metadata
)

var cacheableOpToNameMap = [3]string{"List", "Open", "Metadata"}

// EntryBase implements Entry, making it easy to create new entries.
// You should use plugin.NewEntry to create new EntryBase objects.
type EntryBase struct {
	name string
	// id represents the entry's wash ID. It is set in CachedList.
	id  string
	ttl [3]time.Duration
}

// NewEntry creates a new entry
func NewEntry(name string) EntryBase {
	e := EntryBase{name: name}
	for op := range e.ttl {
		e.SetTTLOf(cacheableOp(op), 15*time.Second)
	}

	return e
}

// Name returns the entry's name.
func (e *EntryBase) Name() string {
	return e.name
}

// ID returns the entry's wash ID
func (e *EntryBase) ID() string {
	return e.id
}

// SetTTLOf sets the specified op's TTL
func (e *EntryBase) SetTTLOf(op cacheableOp, ttl time.Duration) {
	e.ttl[op] = ttl
}

// TurnOffCachingFor turns off caching for the specified op
func (e *EntryBase) TurnOffCachingFor(op cacheableOp) {
	e.SetTTLOf(op, -1)
}

// TurnOffCaching turns off caching for all ops
func (e *EntryBase) TurnOffCaching() {
	for op := range e.ttl {
		e.TurnOffCachingFor(cacheableOp(op))
	}
}

func (e *EntryBase) getTTLOf(op cacheableOp) time.Duration {
	return e.ttl[op]
}

func (e *EntryBase) setID(id string) {
	e.id = id
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

package types

import (
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// Entry represents an Entry as interpreted by `wash find`
type Entry struct {
	apitypes.Entry
	NormalizedPath string
	Metadata       plugin.JSONObject
	SchemaKnown    bool
	Schema         *EntrySchema
}

// NewEntry constructs a new `wash find` entry
func NewEntry(e apitypes.Entry, normalizedPath string) Entry {
	return Entry{
		Entry:          e,
		NormalizedPath: normalizedPath,
		Metadata:       e.Attributes.Meta(),
	}
}

// SetSchema sets the entry's schema. Note that s == nil
// means the entry's schema was pruned from the stree.
func (e *Entry) SetSchema(s *EntrySchema) {
	e.SchemaKnown = true
	e.Schema = s
}

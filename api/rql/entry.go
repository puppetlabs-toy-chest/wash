package rql

import apitypes "github.com/puppetlabs/wash/api/types"

// EntrySchema represents an RQL entry's schema
type EntrySchema = apitypes.EntrySchema

// Entry represents an RQL entry
type Entry struct {
	apitypes.Entry
	Schema *EntrySchema
}

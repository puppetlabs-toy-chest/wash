package rql

import (
	"github.com/getlantern/deepcopy"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// EntrySchema represents an RQL entry's schema
type EntrySchema = apitypes.EntrySchema

func newEntrySchema(s *plugin.EntrySchema) *EntrySchema {
	// apitypes.EntrySchema sets up its graph and children
	// in its Unmarshal method. Thus we marshal s to JSON
	// then unmarshal it as apitypes.EntrySchema. deepcopy.Copy
	// does all of this for us so just use that.
	//
	// TODO: Update apitypes.NewEntrySchema to initialize
	// everything. This should be done once RQL is merged
	// or rebased and we have access to plugin.SchemaGraph
	// (or something similar)
	var schema *apitypes.EntrySchema
	if err := deepcopy.Copy(&schema, s); err != nil {
		panic(err)
	}
	return schema
}

// Entry represents an RQL entry
type Entry struct {
	apitypes.Entry
	Schema      *EntrySchema
	pluginEntry plugin.Entry
}

func newEntry(parent *Entry, pluginEntry plugin.Entry) Entry {
	e := Entry{
		Entry:       apitypes.NewEntry(pluginEntry),
		pluginEntry: pluginEntry,
	}
	if parent == nil {
		// This is the root
		e.Path = ""
	} else if parent.Path == "" {
		// This is a child of the root
		e.Path = e.CName
	} else {
		e.Path = parent.Path + "/" + e.CName
	}
	return e
}

func (e Entry) SchemaKnown() bool {
	return e.Schema != nil
}

package gcp

import (
	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/plugin"
)

type storageObject struct {
	plugin.EntryBase
	storageProjectClient
}

func newStorageObject(client storageProjectClient, object *storage.ObjectAttrs) *storageObject {
	obj := &storageObject{EntryBase: plugin.NewEntry(object.Name), storageProjectClient: client}
	obj.Attributes().SetMeta(object)
	return obj
}

func (s *storageObject) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "object").SetMetaAttributeSchema(storage.ObjectAttrs{})
}

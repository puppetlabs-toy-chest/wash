package gcp

import (
	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/plugin"
)

type storageObject struct {
	plugin.EntryBase
	*storage.ObjectHandle
}

func newStorageObject(name string, object *storage.ObjectHandle, attrs *storage.ObjectAttrs) *storageObject {
	obj := &storageObject{EntryBase: plugin.NewEntry(name), ObjectHandle: object}
	obj.Attributes().SetMtime(attrs.Updated).SetSize(uint64(attrs.Size)).SetMeta(attrs)
	return obj
}

func (s *storageObject) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "object").SetMetaAttributeSchema(storage.ObjectAttrs{})
}

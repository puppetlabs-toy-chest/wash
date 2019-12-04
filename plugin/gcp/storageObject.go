package gcp

import (
	"context"
	"io/ioutil"

	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

type storageObject struct {
	plugin.EntryBase
	*storage.ObjectHandle
}

func newStorageObject(name string, object *storage.ObjectHandle, attrs *storage.ObjectAttrs) *storageObject {
	obj := &storageObject{EntryBase: plugin.NewEntry(name), ObjectHandle: object}
	obj.Attributes().
		SetCrtime(attrs.Created).
		SetCtime(attrs.Updated).
		SetMtime(attrs.Updated).
		SetSize(uint64(attrs.Size)).
		SetMeta(attrs)
	return obj
}

func (s *storageObject) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(s, "object").
		SetDescription(storageObjectDescription).
		SetMetaAttributeSchema(storage.ObjectAttrs{})
}

func (s *storageObject) Read(ctx context.Context, size int64, offset int64) ([]byte, error) {
	rdr, err := s.NewRangeReader(context.Background(), offset, int64(size))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rdr.Close(); err != nil {
			activity.Record(ctx, "Error closing GCP storage object range reader: %v", err)
		}
	}()
	return ioutil.ReadAll(rdr)
}

func (s *storageObject) Delete(ctx context.Context) (bool, error) {
	err := s.ObjectHandle.Delete(ctx)
	return true, err
}

const storageObjectDescription = `
This is a Storage object. See the bucket's description for more details
on why we have this kind of entry.
`

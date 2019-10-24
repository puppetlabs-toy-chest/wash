package gcp

import (
	"context"

	"cloud.google.com/go/storage"
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

func (s *storageObject) Open(ctx context.Context) (plugin.SizedReader, error) {
	return &objectReader{ObjectHandle: s.ObjectHandle, size: int64(s.Attributes().Size())}, nil
}

func (s *storageObject) Delete(ctx context.Context) (bool, error) {
	err := s.ObjectHandle.Delete(ctx)
	return true, err
}

type objectReader struct {
	*storage.ObjectHandle
	size int64
}

// This is a fairly inefficient implementation that always reads exactly what's asked for.
// However most uses will probably read the content once and buffer it themselves.
// TODO: buffer some so that we don't read lots of small chunks.
func (r *objectReader) ReadAt(p []byte, off int64) (int, error) {
	rdr, err := r.NewRangeReader(context.Background(), off, int64(len(p)))
	if err != nil {
		return 0, err
	}
	return rdr.Read(p)
}

func (r *objectReader) Size() int64 {
	return r.size
}

const storageObjectDescription = `
This is a Storage object. See the bucket's description for more details
on why we have this kind of entry.
`

package gcp

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/plugin"
)

type storageBucket struct {
	plugin.EntryBase
}

func newStorageBucket(bucket *storage.BucketAttrs) *storageBucket {
	stor := &storageBucket{plugin.NewEntry(bucket.Name)}
	stor.Attributes().SetMeta(bucket)
	return stor
}

func (s *storageBucket) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "bucket").SetMetaAttributeSchema(storage.BucketAttrs{})
}

func (s *storageBucket) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{}
}

// List all storage objects as dirs and files.
func (s *storageBucket) List(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{}, nil
}

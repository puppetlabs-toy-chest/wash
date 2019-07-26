package gcp

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type storageBucket struct {
	plugin.EntryBase
	storageProjectClient
}

func newStorageBucket(client storageProjectClient, bucket *storage.BucketAttrs) *storageBucket {
	stor := &storageBucket{EntryBase: plugin.NewEntry(bucket.Name), storageProjectClient: client}
	stor.Attributes().SetMeta(bucket)
	return stor
}

// List all storage objects as dirs and files.
func (s *storageBucket) List(ctx context.Context) ([]plugin.Entry, error) {
	bucket := s.Bucket(s.Name())

	var entries []plugin.Entry
	it := bucket.Objects(ctx, nil)
	for {
		objectAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, newStorageObject(s.storageProjectClient, objectAttrs))
	}
	return entries, nil
}

func (s *storageBucket) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "bucket").SetMetaAttributeSchema(storage.BucketAttrs{})
}

func (s *storageBucket) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{(&storageObject{}).Schema()}
}

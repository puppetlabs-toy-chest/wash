package gcp

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/plugin"
)

type storageObjectPrefix struct {
	plugin.EntryBase
	bucket *storage.BucketHandle
	prefix string
}

// Takes the name of the directory, as well as the full prefix path.
// Attrs may be nil if they could not be retrieved. Some prefixes don't appear to have attributes.
func newStorageObjectPrefix(bucket *storage.BucketHandle,
	name, prefix string, attrs *storage.ObjectAttrs) *storageObjectPrefix {
	pre := &storageObjectPrefix{
		EntryBase: plugin.NewEntry(name),
		bucket:    bucket,
		prefix:    prefix,
	}
	if attrs != nil {
		pre.Attributes().
			SetCrtime(attrs.Created).
			SetCtime(attrs.Updated).
			SetMtime(attrs.Updated).
			SetSize(uint64(attrs.Size)).
			SetMeta(attrs)
	}
	return pre
}

// List all storage objects under this prefix as dirs and files.
func (s *storageObjectPrefix) List(ctx context.Context) ([]plugin.Entry, error) {
	return listBucket(ctx, s.bucket, s.prefix)
}

func (s *storageObjectPrefix) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "prefix").
		SetDescription(storageObjectPrefixDescription).
		SetMetaAttributeSchema(storage.ObjectAttrs{})
}

func (s *storageObjectPrefix) ChildSchemas() []*plugin.EntrySchema {
	return bucketSchemas()
}

const storageObjectPrefixDescription = `
This represents a common prefix shared by multiple Storage objects. See
the bucket's description for more details on why we have this kind of entry.
`

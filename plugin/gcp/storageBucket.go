package gcp

import (
	"context"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type storageBucket struct {
	plugin.EntryBase
	storageProjectClient
}

func newStorageBucket(client storageProjectClient, bucket *storage.BucketAttrs) *storageBucket {
	stor := &storageBucket{EntryBase: plugin.NewEntry(bucket.Name), storageProjectClient: client}
	stor.Attributes().
		SetCrtime(bucket.Created).
		SetCtime(bucket.Created).
		SetMtime(bucket.Created).
		SetMeta(bucket)
	return stor
}

// List all storage objects as dirs and files.
func (s *storageBucket) List(ctx context.Context) ([]plugin.Entry, error) {
	bucket := s.Bucket(s.Name())
	return listBucket(ctx, bucket, "")
}

func (s *storageBucket) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "bucket").
		SetMetaAttributeSchema(storage.BucketAttrs{}).
		SetDescription(storageBucketDescription)
}

func (s *storageBucket) ChildSchemas() []*plugin.EntrySchema {
	return bucketSchemas()
}

const delimiter = "/"

func listBucket(ctx context.Context, bucket *storage.BucketHandle, prefix string) ([]plugin.Entry, error) {
	var entries []plugin.Entry
	// Get objects directly under this prefix.
	it := bucket.Objects(ctx, &storage.Query{Delimiter: delimiter, Prefix: prefix})
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// https://godoc.org/cloud.google.com/go/storage#Query notes that providing a delimiter returns
		// results in a directory-like fashion. Results will contain objects whose names, aside from
		// the prefix, do not contain delimiter. Objects whose names, aside from the prefix, contain
		// delimiter will have their name, truncated after the delimiter, returned in prefixes.
		// Duplicate prefixes are omitted, and if Prefix is filled in then no other attributes are
		// included.
		if objAttrs.Prefix != "" {
			name := strings.TrimPrefix(strings.TrimSuffix(objAttrs.Prefix, delimiter), prefix)
			preAttrs, err := bucket.Object(objAttrs.Prefix).Attrs(ctx)
			if err != nil {
				// Don't treat this as an error. Not all prefixes have attributes.
				activity.Record(ctx, "Could not get attributes of %v: %v", objAttrs.Prefix, err)
			}
			entries = append(entries, newStorageObjectPrefix(bucket, name, objAttrs.Prefix, preAttrs))
		} else if objAttrs.Name != prefix {
			name := strings.TrimPrefix(objAttrs.Name, prefix)
			entries = append(entries, newStorageObject(name, bucket.Object(objAttrs.Name), objAttrs))
		}
	}
	return entries, nil
}

func bucketSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{(&storageObjectPrefix{}).Schema(), (&storageObject{}).Schema()}
}

const storageBucketDescription = `
This is a Storage bucket. For convenience, we impose some hierarchical structure
on its objects by grouping keys with common prefixes into a specific directory.
For example, the objects 'foo/bar' and 'foo/baz' are represented as files with
path 'foo/bar' and path 'foo/baz', where 'foo' is represented as a 'directory'.
Thus, if you ls this bucket, then everything you'll see is either a Storage
object prefix ('directory') or a Storage object ('file').
`

package gcp

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/puppetlabs/wash/plugin"
)

type firestoreDir struct {
	plugin.EntryBase
	client *firestore.Client
}

func newFirestoreDir(ctx context.Context, projID string) (*firestoreDir, error) {
	cli, err := firestore.NewClient(context.Background(), projID)
	if err != nil {
		return nil, err
	}
	f := &firestoreDir{
		EntryBase: plugin.NewEntry("firestore"),
		client:    cli,
	}
	if _, err := plugin.List(ctx, f); err != nil {
		f.MarkInaccessible(ctx, err)
	}
	return f, nil
}

// List all collections as dirs
func (f *firestoreDir) List(ctx context.Context) ([]plugin.Entry, error) {
	colls, err := f.client.Collections(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	return toCollectionEntries(f.client, "", colls), nil
}

func (f *firestoreDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(f, "firestore").
		IsSingleton().
		SetDescription(firestoreDirDescription)
}

func (f *firestoreDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&firestoreCollection{}).Schema(),
	}
}

func toCollectionEntries(client *firestore.Client, parent string, colls []*firestore.CollectionRef) []plugin.Entry {
	entries := make([]plugin.Entry, len(colls))
	for ix, coll := range colls {
		entries[ix] = newFirestoreCollection(client, parent, coll)
	}
	return entries
}

func firestorePath(parent string, id string) string {
	if len(parent) <= 0 {
		return id
	}
	return parent + "/" + id
}

const firestoreDirDescription = `
This directory represents the firestore database. Its entries consist of collections
and documents.

You can view a document's data by cat'ing its corresponding data.json file, OR by
viewing the "data" key in the document's metadata. The latter's useful because it
lets you use find's 'meta' primary to filter documents on their data values. For
example, something like

  find <collection> -meta '.data.foo' 5	

will return all documents in <collection> whose 'foo' field is equal to 5.

NOTE: Filtering with find does not (yet) take advantage of Firestore queries. If you
need that optimization, then please file an issue!
`

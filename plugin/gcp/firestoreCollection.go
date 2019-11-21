package gcp

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/puppetlabs/wash/plugin"
)

type firestoreCollection struct {
	plugin.EntryBase
	client *firestore.Client
	path   string
}

func newFirestoreCollection(client *firestore.Client, parent string, collRef *firestore.CollectionRef) *firestoreCollection {
	return &firestoreCollection{
		EntryBase: plugin.NewEntry(collRef.ID),
		client:    client,
		path:      firestorePath(parent, collRef.ID),
	}
}

func (coll *firestoreCollection) List(ctx context.Context) ([]plugin.Entry, error) {
	docs, err := coll.client.Collection(coll.path).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	entries := make([]plugin.Entry, len(docs))
	for ix, doc := range docs {
		entries[ix] = newFirestoreDocument(coll.client, coll.path, doc)
	}
	return entries, nil
}

func (coll *firestoreCollection) Delete(ctx context.Context) (bool, error) {
	// According to https://stackoverflow.com/a/47861164, deleting a collection
	// means deleting its documents.
	batch := coll.client.Batch()
	docs, err := coll.client.Collection(coll.path).DocumentRefs(ctx).GetAll()
	if err != nil {
		return false, err
	}
	for _, doc := range docs {
		batch.Delete(doc)
	}
	_, err = batch.Commit(ctx)
	return true, err
}

func (coll *firestoreCollection) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(coll, "collection").
		SetDescription(firestoreCollectionDescription)
}

func (coll *firestoreCollection) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&firestoreDocument{}).Schema(),
	}
}

const firestoreCollectionDescription = `
This is a Firestore collection. See the 'firestore' directory's docs for
more details on why we have this kind of entry.
`

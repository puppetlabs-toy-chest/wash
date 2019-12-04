package gcp

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/puppetlabs/wash/plugin"
)

type firestoreDocument struct {
	plugin.EntryBase
	client *firestore.Client
	path   string
	data   map[string]interface{}
}

type firestoreDocumentMetadata struct {
	CreateTime time.Time              `json:"CreateTime"`
	UpdateTime time.Time              `json:"UpdateTime"`
	ReadTime   time.Time              `json:"ReadTime"`
	Data       map[string]interface{} `json:"Data"`
}

func newFirestoreDocument(client *firestore.Client, parent string, snapshot *firestore.DocumentSnapshot) *firestoreDocument {
	doc := &firestoreDocument{
		EntryBase: plugin.NewEntry(snapshot.Ref.ID),
		client:    client,
		path:      firestorePath(parent, snapshot.Ref.ID),
		data:      snapshot.Data(),
	}

	doc.Attributes().
		SetCrtime(snapshot.CreateTime).
		SetCtime(snapshot.UpdateTime).
		SetMtime(snapshot.UpdateTime).
		SetMeta(firestoreDocumentMetadata{
			snapshot.CreateTime,
			snapshot.UpdateTime,
			snapshot.ReadTime,
			snapshot.Data(),
		})

	return doc
}

func (doc *firestoreDocument) List(ctx context.Context) ([]plugin.Entry, error) {
	colls, err := doc.client.Doc(doc.path).Collections(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	dataJSON, err := newFirestoreDocumentDataJSON(doc.data)
	if err != nil {
		return nil, err
	}
	collEntries := toCollectionEntries(doc.client, doc.path, colls)
	return append([]plugin.Entry{dataJSON}, collEntries...), nil
}

func (doc *firestoreDocument) Delete(ctx context.Context) (bool, error) {
	_, err := doc.client.Doc(doc.path).Delete(ctx)
	return true, err
}

func (doc *firestoreDocument) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(doc, "document").
		SetMetaAttributeSchema(firestoreDocumentMetadata{}).
		SetDescription(firestoreDocumentDescription)
}

func (doc *firestoreDocument) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&firestoreDocumentDataJSON{}).Schema(),
		(&firestoreCollection{}).Schema(),
	}
}

const firestoreDocumentDescription = `
This is a Firestore document. See the 'firestore' directory's docs for
more details on why we have this kind of entry.
`

type firestoreDocumentDataJSON struct {
	plugin.EntryBase
	bytes []byte
}

func newFirestoreDocumentDataJSON(data map[string]interface{}) (*firestoreDocumentDataJSON, error) {
	dataBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// This should never happen
		return nil, err
	}
	dataEntry := &firestoreDocumentDataJSON{
		EntryBase: plugin.NewEntry("data.json"),
		bytes:     dataBytes,
	}
	dataEntry.DisableDefaultCaching()
	return dataEntry, nil
}

func (data *firestoreDocumentDataJSON) Read(ctx context.Context) ([]byte, error) {
	return data.bytes, nil
}

func (data *firestoreDocumentDataJSON) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(data, "data.json").
		IsSingleton().
		SetDescription(firestoreDocumentDataJSONDescription)
}

const firestoreDocumentDataJSONDescription = `
This is a Firestore document's data as pretty-printed JSON. See the
'firestore' directory's docs for more details on why we have this
kind of entry.
`

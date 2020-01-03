package plugin

import (
	"context"
	"encoding/json"
)

// MetadataJSONFile represents a metadata.json file that contains another entry's metadata.
type MetadataJSONFile struct {
	EntryBase
	other Entry
}

// NewMetadataJSONFile creates a new MetadataJSONFile. If caching Metadata on the `other` entry is
// disabled, it will use that to compute the file size upfront.
func NewMetadataJSONFile(ctx context.Context, other Entry) (*MetadataJSONFile, error) {
	return &MetadataJSONFile{
		EntryBase: NewEntry("metadata.json"),
		other:     other,
	}, nil
}

// Schema defines the schema of a metadata.json file.
func (m *MetadataJSONFile) Schema() *EntrySchema {
	return NewEntrySchema(m, "metadata.json").
		SetDescription(metadataJSONDescription).
		IsSingleton()
}

// Read returns the metadata of the `other` entry as its content.
func (m *MetadataJSONFile) Read(ctx context.Context) ([]byte, error) {
	meta, err := Metadata(ctx, m.other)
	if err != nil {
		return nil, err
	}

	prettyMeta, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}

	return prettyMeta, nil
}

const metadataJSONDescription = `
A read-only 'file' whose content contains the underlying entry's full metadata.
This makes it easier for you to grep its values.
`

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
	meta := &MetadataJSONFile{
		EntryBase: NewEntry("metadata.json"),
		other:     other,
	}

	if other.getTTLOf(MetadataOp) < 0 {
		// Content is presumably easy to get, so use it to determine size.
		content, err := meta.Read(ctx)
		if err != nil {
			return nil, err
		}

		meta.Attributes().SetSize(uint64(len(content)))
	}

	return meta, nil
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

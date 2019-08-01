package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type basicEntry struct {
	EntryBase
}

func (e *basicEntry) Schema() *EntrySchema {
	return NewEntrySchema(e, "basic")
}

func TestMetadataJSONFile(t *testing.T) {
	basic := basicEntry{NewEntry("foo")}
	inst, err := NewMetadataJSONFile(context.Background(), &basic)
	assert.NoError(t, err)
	assert.Equal(t, "metadata.json", inst.Name())
	assert.Implements(t, (*Readable)(nil), inst)
}

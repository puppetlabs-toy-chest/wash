package volume

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

func TestVolumeDir(t *testing.T) {
	dmap, err := StatParseAll(strings.NewReader(fixture), mountpoint)
	assert.Nil(t, err)
	contentcb := func(ctx context.Context, path string) (plugin.SizedReader, error) {
		return nil, nil
	}

	assert.NotNil(t, dmap[""]["path"])
	vd := NewDir("path", dmap[""]["path"], contentcb, "/path", dmap)
	assert.Equal(t, 0755|os.ModeDir, vd.Attr().Mode)

	assert.NotNil(t, dmap[""]["path1"])
	vd = NewDir("path", dmap[""]["path1"], contentcb, "/path1", dmap)
	entries, err := vd.List(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "a file", entries[0].Name())
	if entry, ok := entries[0].(*File); assert.Equal(t, true, ok) {
		assert.Equal(t, "/path1/a file", entry.path)
	}

	assert.NotNil(t, dmap[""]["path2"])
	vd = NewDir("path", dmap[""]["path2"], contentcb, "/path2", dmap)
	entries, err = vd.List(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "dir", entries[0].Name())
	if entry, ok := entries[0].(*Dir); assert.Equal(t, true, ok) {
		assert.Equal(t, "/path2/dir", entry.path)
	}
}

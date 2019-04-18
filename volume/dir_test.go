package volume

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

func TestVolumeDir(t *testing.T) {
	dmap, err := StatParseAll(strings.NewReader(fixture), mountpoint)
	assert.Nil(t, err)
	listcb := func(ctx context.Context) (DirMap, error) {
		return dmap, nil
	}
	contentcb := func(ctx context.Context, path string) (plugin.SizedReader, error) {
		return nil, nil
	}

	plugin.SetTestCache(datastore.NewMemCache())
	entry := plugin.NewEntry("mine")
	entry.SetTestID("/mine")

	assert.NotNil(t, dmap[""]["path"])
	vd := NewDir("path", dmap[""]["path"], &entry, listcb, contentcb, "/path")
	attr := plugin.Attributes(vd)
	assert.Equal(t, 0755|os.ModeDir, attr.Mode())

	assert.NotNil(t, dmap[""]["path1"])
	vd = NewDir("path", dmap[""]["path1"], &entry, listcb, contentcb, "/path1")
	entries, err := vd.List(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "a file", plugin.Name(entries[0]))
	if entry, ok := entries[0].(*File); assert.Equal(t, true, ok) {
		assert.Equal(t, "/path1/a file", entry.path)
	}

	assert.NotNil(t, dmap[""]["path2"])
	vd = NewDir("path", dmap[""]["path2"], &entry, listcb, contentcb, "/path2")
	entries, err = vd.List(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "dir", plugin.Name(entries[0]))
	if entry, ok := entries[0].(*Dir); assert.Equal(t, true, ok) {
		assert.Equal(t, "/path2/dir", entry.path)
	}

	plugin.UnsetTestCache()
}

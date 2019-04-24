package volume

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

type mockDirEntry struct {
	plugin.EntryBase
	dmap DirMap
}

func (m *mockDirEntry) VolumeList(context.Context) (DirMap, error) {
	return m.dmap, nil
}

func (m *mockDirEntry) VolumeOpen(context.Context, string) (plugin.SizedReader, error) {
	return nil, nil
}

func (m *mockDirEntry) VolumeStream(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}

func TestVolumeDir(t *testing.T) {
	dmap, err := StatParseAll(strings.NewReader(fixture), mountpoint)
	assert.Nil(t, err)

	plugin.SetTestCache(datastore.NewMemCache())
	entry := mockDirEntry{EntryBase: plugin.NewEntry("mine"), dmap: dmap}
	entry.SetTestID("/mine")

	assert.NotNil(t, dmap[""]["path"])
	vd := newDir("path", dmap[""]["path"], &entry, "/path")
	attr := plugin.Attributes(vd)
	assert.Equal(t, 0755|os.ModeDir, attr.Mode())

	assert.NotNil(t, dmap[""]["path1"])
	vd = newDir("path", dmap[""]["path1"], &entry, "/path1")
	entries, err := vd.List(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "a file", plugin.Name(entries[0]))
	if entry, ok := entries[0].(*file); assert.Equal(t, true, ok) {
		assert.Equal(t, "/path1/a file", entry.path)
	}

	assert.NotNil(t, dmap[""]["path2"])
	vd = newDir("path", dmap[""]["path2"], &entry, "/path2")
	entries, err = vd.List(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, "dir", plugin.Name(entries[0]))
	if entry, ok := entries[0].(*dir); assert.Equal(t, true, ok) {
		assert.Equal(t, "/path2/dir", entry.path)
	}

	plugin.UnsetTestCache()
}

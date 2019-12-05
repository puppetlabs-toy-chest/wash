package apifs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/puppetlabs/wash/plugin"
)

type dir struct {
	fsnode
}

func newDir(finfo os.FileInfo, path string) *dir {
	d := &dir{
		newFSNode(finfo, path),
	}
	d.DisableDefaultCaching()
	return d
}

func (d *dir) List(ctx context.Context) ([]plugin.Entry, error) {
	matches, err := filepath.Glob(filepath.Join(d.path, "*"))
	if err != nil {
		return nil, err
	}

	entries := make([]plugin.Entry, len(matches))
	for i, match := range matches {
		entry, err := NewEntry(ctx, match)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

func (d *dir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&dir{}).Schema(),
		(&file{}).Schema(),
	}
}

func (d *dir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(d, "dir")
}

var _ = plugin.Parent(&dir{})

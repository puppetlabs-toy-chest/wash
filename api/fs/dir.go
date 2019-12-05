package apifs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/puppetlabs/wash/plugin"
)

type dir struct {
	*fsnode
}

func newDir(ctx context.Context, finfo os.FileInfo, path string) *dir {
	return &dir{
		newFSNode(ctx, finfo, path),
	}
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
	return nil
}

func (d *dir) Schema() *plugin.EntrySchema {
	// Schema only makes sense for core plugins. apifs isn't a core
	// plugin.
	return nil
}

var _ = plugin.Parent(&dir{})

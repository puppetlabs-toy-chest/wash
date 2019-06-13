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

func dirBase() *dir {
	d := &dir{
		fsnode: fsnodeBase(),
	}
	d.SetLabel("dir")
	return d
}

func newDir(ctx context.Context, finfo os.FileInfo, path string) *dir {
	d := dirBase()
	d.build(finfo, path)
	return d
}

func (d *dir) ChildSchemas() []*plugin.EntrySchema {
	return plugin.ChildSchemas(dirBase(), fileBase())
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

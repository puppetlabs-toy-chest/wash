package volume

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

// Dir represents a directory in a volume. It retains access to a map of directories
// to their children and attribute data to populate subdirectories.
type Dir struct {
	plugin.EntryBase
	attr      plugin.Attributes
	contentcb ContentCB
	path      string
	dirs      DirMap
}

// NewDir creates a Dir populated from dirs.
func NewDir(name string, attr plugin.Attributes, cb ContentCB, path string, dirs DirMap) *Dir {
	return &Dir{EntryBase: plugin.NewEntry(name), attr: attr, contentcb: cb, path: path, dirs: dirs}
}

// Attr returns the attributes of the directory.
func (v *Dir) Attr() plugin.Attributes {
	return v.attr
}

// List lists the children of the directory.
func (v *Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	root := v.dirs[v.path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode.IsDir() {
			entries = append(entries, NewDir(name, attr, v.contentcb, v.path+"/"+name, v.dirs))
		} else {
			entries = append(entries, NewFile(name, attr, v.contentcb, v.path+"/"+name))
		}
	}
	return entries, nil
}

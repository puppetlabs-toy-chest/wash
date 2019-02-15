package os

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

// VolumeDir represents a directory in a volume. It retains access to a map of directories
// to their children and attribute data to populate subdirectories.
type VolumeDir struct {
	plugin.EntryBase
	attr      plugin.Attributes
	contentcb ContentCB
	path      string
	dirs      DirMap
}

// NewVolumeDir creates a VolumeDir.
func NewVolumeDir(name string, attr plugin.Attributes, cb ContentCB, path string, dirs DirMap) *VolumeDir {
	return &VolumeDir{EntryBase: plugin.NewEntry(name), attr: attr, contentcb: cb, path: path, dirs: dirs}
}

// Attr returns the attributes of the directory.
func (v *VolumeDir) Attr() plugin.Attributes {
	return v.attr
}

// LS lists the children of the directory.
func (v *VolumeDir) LS(ctx context.Context) ([]plugin.Entry, error) {
	root := v.dirs[v.path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode.IsDir() {
			entries = append(entries, NewVolumeDir(name, attr, v.contentcb, v.path+"/"+name, v.dirs))
		} else {
			entries = append(entries, NewVolumeFile(name, attr, v.contentcb, v.path+"/"+name))
		}
	}
	return entries, nil
}

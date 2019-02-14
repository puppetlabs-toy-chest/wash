package plugin

import (
	"context"
)

// VolumeDir represents a directory in a volume. It retains access to a map of directories
// to their children and attribute data to populate subdirectories.
type VolumeDir struct {
	EntryBase
	attr      Attributes
	contentcb ContentCB
	path      string
	dirs      DirMap
}

// NewVolumeDir creates a VolumeDir.
func NewVolumeDir(name string, attr Attributes, cb ContentCB, path string, dirs DirMap) *VolumeDir {
	return &VolumeDir{EntryBase: NewEntry(name), attr: attr, contentcb: cb, path: path, dirs: dirs}
}

// Attr returns the attributes of the directory.
func (v *VolumeDir) Attr() Attributes {
	return v.attr
}

// LS lists the children of the directory.
func (v *VolumeDir) LS(ctx context.Context) ([]Entry, error) {
	root := v.dirs[v.path]
	entries := make([]Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode.IsDir() {
			entries = append(entries, NewVolumeDir(name, attr, v.contentcb, v.path+"/"+name, v.dirs))
		} else {
			entries = append(entries, NewVolumeFile(name, attr, v.contentcb, v.path+"/"+name))
		}
	}
	return entries, nil
}

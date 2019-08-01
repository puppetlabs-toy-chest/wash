package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// dir represents a directory in a volume. It populates a subtree from the Interface as needed.
type dir struct {
	plugin.EntryBase
	impl   Interface
	path   string
	dirmap DirMap
}

// newDir creates a dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, path string) *dir {
	vd := &dir{
		EntryBase: plugin.NewEntry(name),
	}
	vd.impl = impl
	vd.path = path
	vd.SetAttributes(attr)
	vd.SetTTLOf(plugin.ListOp, 30*time.Second)
	return vd
}

func (v *dir) ChildSchemas() []*plugin.EntrySchema {
	return ChildSchemas()
}

func (v *dir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(v, "dir").SetEntryType("volumeDir")
}

// Generate children using the provided DirMap. The dir may not have a dirmap
// stored if it's a source because it should dynamically generate it.
func (v *dir) generateChildren(dirmap DirMap) []plugin.Entry {
	parent := dirmap[v.path]
	entries := make([]plugin.Entry, 0, len(parent))
	for name, attr := range parent {
		subpath := v.path + "/" + name
		if attr.Mode().IsDir() {
			newEntry := newDir(name, attr, v.impl, subpath)
			if d, ok := dirmap[subpath]; ok && d != nil {
				newEntry.dirmap = dirmap
				newEntry.Prefetched()
			}
			entries = append(entries, newEntry)
		} else {
			entries = append(entries, newFile(name, attr, v.impl, subpath))
		}
	}
	return entries
}

// List lists the children of the directory.
func (v *dir) List(ctx context.Context) ([]plugin.Entry, error) {
	if v.dirmap != nil {
		// Children have been pre-populated by a source parent.
		return v.generateChildren(v.dirmap), nil
	}

	// Generate child hierarchy. Don't store it on this entry, but populate new dirs from it.
	dirmap, err := v.impl.VolumeList(ctx, v.path)
	if err != nil {
		return nil, err
	}

	return v.generateChildren(dirmap), nil
}

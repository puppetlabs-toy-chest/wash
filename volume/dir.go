package volume

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

// dir represents a directory in a volume. It populates a subtree from the Interface as needed.
type dir struct {
	plugin.EntryBase
	impl   Interface
	path   string
	dirmap *dirMap
}

// newDir creates a dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, path string) *dir {
	vd := &dir{
		EntryBase: plugin.NewEntry(name),
	}
	vd.impl = impl
	vd.path = path
	vd.SetAttributes(attr)
	return vd
}

func (v *dir) ChildSchemas() []*plugin.EntrySchema {
	return ChildSchemas()
}

func (v *dir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(v, "dir").SetDescription(dirDescription)
}

// Generate children using the provided DirMap. The dir may not have a dirmap
// stored if it's a source because it should dynamically generate it.
func (v *dir) generateChildren(dirmap *dirMap) []plugin.Entry {
	dirmap.mux.RLock()
	defer dirmap.mux.RUnlock()

	parent := dirmap.mp[v.path]
	entries := make([]plugin.Entry, 0, len(parent))
	for name, attr := range parent {
		subpath := v.path + "/" + name
		if attr.Mode().IsDir() {
			newEntry := newDir(name, attr, v.impl, subpath)
			newEntry.SetTTLOf(plugin.ListOp, ListTTL)
			if d, ok := dirmap.mp[subpath]; ok && d != nil {
				newEntry.dirmap = dirmap
				newEntry.Prefetched()
				newEntry.DisableCachingFor(plugin.ListOp)
			}
			entries = append(entries, newEntry)
		} else {
			newEntry := newFile(name, attr, v.impl, subpath)
			newEntry.dirmap = dirmap
			entries = append(entries, newEntry)
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

	return v.generateChildren(&dirMap{mp: dirmap}), nil
}

func (v *dir) Delete(ctx context.Context) (bool, error) {
	return deleteNode(ctx, v.impl, v.path, v.dirmap)
}

const dirDescription = `
This is a directory on a remote volume or a container/VM.
`

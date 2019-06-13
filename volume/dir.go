package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// dir represents a directory in a volume. It populates a subtree from the Interface as needed.
type dir struct {
	plugin.EntryBase
	impl Interface
	key  plugin.Entry
	path string
}

func dirBase() *dir {
	vd := &dir{
		EntryBase: plugin.NewEntryBase(),
	}
	vd.SetLabel("dir")
	return vd
}

// newDir creates a dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, key plugin.Entry, path string) *dir {
	vd := dirBase()
	vd.impl = impl
	vd.key = key
	vd.path = path
	vd.SetName(name)
	vd.SetAttributes(attr)
	vd.SetTTLOf(plugin.OpenOp, 60*time.Second)
	// Caching handled in List based on 'impl'.
	vd.DisableCachingFor(plugin.ListOp)

	return vd
}

func (v *dir) ChildSchemas() []*plugin.EntrySchema {
	return ChildSchemas()
}

// List lists the children of the directory.
func (v *dir) List(ctx context.Context) ([]plugin.Entry, error) {
	result, err := plugin.CachedOp(ctx, "VolumeListCB", v.key, 30*time.Second, func() (interface{}, error) {
		return v.impl.VolumeList(ctx, v.path)
	})
	if err != nil {
		return nil, err
	}

	root := result.(DirMap)[v.path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode().IsDir() {
			subpath := v.path + "/" + name
			newEntry := newDir(name, attr, v.impl, v.key, subpath)
			if d, ok := result.(DirMap)[subpath]; !ok || d == nil {
				// Update key so we trigger new exploration with a new cache key at this subpath.
				newEntry.key = newEntry
			}
			entries = append(entries, newEntry)
		} else {
			entries = append(entries, newFile(name, attr, v.impl, v.path+"/"+name))
		}
	}
	return entries, nil
}

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
	// subtreeRoot represents the location we started searching for this subtree, which is used both as
	// a cache key for that search and to be able to repeat the search. We must always query the same
	// path when running VolumeList for its related key. If we didn't, we might start with a cache at
	// '/', then later refill the same cache entry with a hierarchy starting at '/foo'. If we used that
	// new cache data to try and list '/', we'd only get back a directory containing 'foo' and omit any
	// other files in '/' because they wouldn't be in the cache at the time.
	subtreeRoot *dir
	path        string
}

func dirBase() *dir {
	vd := &dir{
		EntryBase: plugin.NewEntryBase(),
	}
	vd.SetLabel("dir")
	return vd
}

// newDir creates a dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, subtreeRoot *dir, path string) *dir {
	vd := dirBase()
	vd.impl = impl
	vd.subtreeRoot = subtreeRoot
	if vd.subtreeRoot == nil {
		vd.subtreeRoot = vd
	}
	vd.path = path
	vd.SetName(name)
	vd.SetAttributes(attr)
	vd.SetTTLOf(plugin.OpenOp, 60*time.Second)
	// Caching handled in List based on 'impl'.
	vd.DisableCachingFor(plugin.ListOp)

	return vd
}

func (v *dir) ChildSchemas() []plugin.EntrySchema {
	return ChildSchemas()
}

// List lists the children of the directory.
func (v *dir) List(ctx context.Context) ([]plugin.Entry, error) {
	// Use subtree root if specified. If it's the base path, then use impl instead because we started
	// from a dummy root that doesn't have an ID, so can't be used for caching.
	var subtreeRoot plugin.Entry = v.subtreeRoot
	if v.subtreeRoot.path == RootPath {
		subtreeRoot = v.impl
	}
	result, err := plugin.CachedOp(ctx, "VolumeListCB", subtreeRoot, 30*time.Second, func() (interface{}, error) {
		return v.impl.VolumeList(ctx, v.subtreeRoot.path)
	})
	if err != nil {
		return nil, err
	}

	root := result.(DirMap)[v.path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode().IsDir() {
			subpath := v.path + "/" + name
			newEntry := newDir(name, attr, v.impl, v.subtreeRoot, subpath)
			if d, ok := result.(DirMap)[subpath]; !ok || d == nil {
				// Update key so we trigger new exploration with a new cache key at this subpath.
				newEntry.subtreeRoot = newEntry
			}
			entries = append(entries, newEntry)
		} else {
			entries = append(entries, newFile(name, attr, v.impl, v.path+"/"+name))
		}
	}
	return entries, nil
}

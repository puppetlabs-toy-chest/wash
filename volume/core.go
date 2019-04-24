// Package volume provides helpers for representing a remote filesystem.
//
// Plugins should use these helpers when representing a filesystem where the
// structure and stats are retrieved all-at-once. The filesystem representation
// should be stored in 'DirMap'. The root of the filesystem is then created with
// 'NewDir'.
package volume

import (
	"context"
	"sort"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// Interface presents methods to access the volume.
//
// Method names for this interface are chosen to make it simple to distinguish them from
// methods implemented to satisfy plugin interfaces.
type Interface interface {
	plugin.Entry

	// Returns a map of volume nodes to their stats, such as that returned by StatParseAll.
	VolumeList(context.Context) (DirMap, error)
	// Accepts a path and returns the content associated with that path.
	VolumeOpen(context.Context, string) (plugin.SizedReader, error)
	// TODO: add VolumeStream
}

// A Dir is a map of files in a directory to their attributes.
type Dir = map[string]plugin.EntryAttributes

// A DirMap is a map of directory names to a map of their directory contents.
type DirMap = map[string]Dir

// List constructs an array of entries for the given path from a DirMap.
// The root path is an empty string. Requests are cached against the supplied Interface
// using the VolumeListCB op.
func List(ctx context.Context, impl Interface, path string) ([]plugin.Entry, error) {
	result, err := plugin.CachedOp(ctx, "VolumeListCB", impl, 30*time.Second, func() (interface{}, error) {
		return impl.VolumeList(ctx)
	})
	if err != nil {
		return nil, err
	}

	root := result.(DirMap)[path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode().IsDir() {
			entries = append(entries, newDir(name, attr, impl, path+"/"+name))
		} else {
			entries = append(entries, newFile(name, attr, impl.VolumeOpen, path+"/"+name))
		}
	}
	// Sort entries so they have a deterministic order.
	sort.Slice(entries, func(i, j int) bool { return plugin.Name(entries[i]) < plugin.Name(entries[j]) })
	return entries, nil
}

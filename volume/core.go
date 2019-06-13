// Package volume provides helpers for representing a remote filesystem.
//
// Plugins should use these helpers when representing a filesystem where the
// structure and stats are retrieved all-at-once. The filesystem representation
// should be stored in 'DirMap'. The root of the filesystem is then created with
// 'NewDir'.
package volume

import (
	"context"
	"io"
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
	// DirMap must include items starting from the specified path. If an entry in DirMap
	// points to a nil Dir, it is assumed to be unexplored.
	VolumeList(ctx context.Context, path string) (DirMap, error)
	// Accepts a path and returns the content associated with that path.
	VolumeOpen(ctx context.Context, path string) (plugin.SizedReader, error)
	// Accepts a path and streams updates to the content associated with that path.
	VolumeStream(ctx context.Context, path string) (io.ReadCloser, error)
}

// A Dir is a map of files in a directory to their attributes.
type Dir = map[string]plugin.EntryAttributes

// A DirMap is a map of directory names to a map of their directory contents.
// The Dir may be a nil map, in which case we assume that its children have not been
// discovered yet and will run VolumeList on the directory.
type DirMap = map[string]Dir

// ChildSchemas returns a volume's child schema
func ChildSchemas() []plugin.EntrySchema {
	return plugin.ChildSchemas(dirBase(), fileBase())
}

// List constructs an array of entries for the given path from a DirMap.
// The root path is an empty string. If a directory that hasn't been explored yet is listed it
// will conduct further exploration. Requests are cached against the supplied Interface using the
// VolumeListCB op. The supplied impl and path must have a 1-to-1 association.
func List(ctx context.Context, impl Interface, path string) ([]plugin.Entry, error) {
	result, err := plugin.CachedOp(ctx, "VolumeListCB", impl, 30*time.Second, func() (interface{}, error) {
		return impl.VolumeList(ctx, path)
	})
	if err != nil {
		return nil, err
	}

	root := result.(DirMap)[path]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode().IsDir() {
			subpath := path + "/" + name
			newEntry := newDir(name, attr, impl, subpath)
			if d, ok := result.(DirMap)[subpath]; !ok || d == nil {
				entries = append(entries, &unDir{newEntry})
			} else {
				entries = append(entries, newEntry)
			}
		} else {
			entries = append(entries, newFile(name, attr, impl, path+"/"+name))
		}
	}
	return entries, nil
}

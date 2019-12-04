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
	"strings"
	"sync"
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
	VolumeRead(ctx context.Context, path string) ([]byte, error)
	// Accepts a path and streams updates to the content associated with that path.
	VolumeStream(ctx context.Context, path string) (io.ReadCloser, error)
	// Deletes the volume node at the specified path. Mirrors plugin.Deletable#Delete
	VolumeDelete(ctx context.Context, path string) (bool, error)
}

// Children represents a directory's children. It is a map of <child_basename> => <child_attributes>.
type Children = map[string]plugin.EntryAttributes

// A DirMap is a map of <dir_path> => <children>. If <children> is nil, then that means
// that <dir_path>'s children haven't been discovered yet, so we will run VolumeList on
// <dir_path>.
type DirMap = map[string]Children

// dirMap is a thread-safe wrapper to a DirMap object. It is needed to properly implement
// Delete.
type dirMap struct {
	mp  DirMap
	mux sync.RWMutex
}

// ChildSchemas returns a volume's child schema
func ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&dir{}).Schema(),
		(&file{}).Schema(),
	}
}

// RootPath is the root of the filesystem described by a DirMap returned from VolumeList.
const RootPath = ""

// List constructs an array of entries for the given path from a DirMap.
// If a directory that hasn't been explored yet is listed it will conduct further exploration.
// Requests are cached against the supplied Interface using the VolumeListCB op.
func List(ctx context.Context, impl Interface) ([]plugin.Entry, error) {
	// Start with the implementation as the cache key so we re-use data we get from it for subdirectory queries.
	return newDir("dummy", plugin.EntryAttributes{}, impl, RootPath).List(ctx)
}

// ListTTL represents the List op's TTL. The entry implementing volume.Interface should
// set the List op's TTL to this value.
const ListTTL = 30 * time.Second

// delete is a keyword, so we use deleteNode instead. Note that this implementation is
// symmetric with plugin.Delete except that we are managing a dirmap instead of a cache.
func deleteNode(ctx context.Context, impl Interface, path string, dirmap *dirMap) (deleted bool, err error) {
	deleted, err = impl.VolumeDelete(ctx, path)
	if err != nil {
		return
	}
	if !deleted {
		return
	}

	// The node was deleted so remove it from the dirmap and from its parent's children
	dirmap.mux.Lock()
	defer dirmap.mux.Unlock()

	delete(dirmap.mp, path)
	segments := strings.Split(path, "/")
	parentPath := strings.Join(segments[:len(segments)-1], "/")
	if parentChildren, ok := dirmap.mp[parentPath]; ok {
		basename := segments[len(segments)-1]
		delete(parentChildren, basename)
	}
	return
}

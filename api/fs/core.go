// Package apifs is used by the Wash API to convert local files/directories
// into Wash entries
package apifs

import (
	"os"

	"github.com/puppetlabs/wash/plugin"
)

// fsnode => filesystem node
type fsnode struct {
	plugin.EntryBase
	path string
}

func newFSNode(finfo os.FileInfo, path string) *fsnode {
	// TODO: finfo.Sys() contains more detailed file attributes,
	// but it's platform-specific. We should eventually use it for
	// a more complete implementation of apifs.
	attr := plugin.EntryAttributes{}
	attr.
		SetMtime(finfo.ModTime()).
		SetMode(finfo.Mode()).
		SetSize(uint64(finfo.Size())).
		SetMeta(plugin.ToMeta(finfo))

	n := &fsnode{
		EntryBase: plugin.NewEntry(finfo.Name()),
		path:      path,
	}
	n.DisableDefaultCaching()
	n.SetAttributes(attr)
	return n
}

// NewEntry constructs a new Wash entry from the given FS
// path
func NewEntry(path string) (plugin.Entry, error) {
	finfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if finfo.IsDir() {
		return newDir(finfo, path), nil
	}
	return newFile(finfo, path), nil
}

// Package apifs is used by the Wash API to convert local files/directories
// into Wash entries
package apifs

import (
	"context"
	"os"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// fsnode => filesystem node
type fsnode struct {
	plugin.EntryBase
	path string
}

func newFSNode(ctx context.Context, finfo os.FileInfo, path string) *fsnode {
	// TODO: finfo.Sys() contains more detailed file attributes,
	// but it's platform-specific. We should eventually use it for
	// a more complete implementation of apifs.
	attr := plugin.EntryAttributes{}
	attr.
		SetMtime(finfo.ModTime()).
		SetMode(finfo.Mode()).
		SetSize(uint64(finfo.Size())).
		SetMeta(plugin.ToMeta(newFileInfo(finfo)))

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
func NewEntry(ctx context.Context, path string) (plugin.Entry, error) {
	finfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if finfo.IsDir() {
		return newDir(ctx, finfo, path), nil
	}
	return newFile(ctx, finfo, path), nil
}

// os.FileInfo is an interface. Its implementations do not export
// any fields, so we cannot directly marshal an os.FileInfo object
// into JSON. Thus, we create a wrapper "fileInfo" object whose fields
// map to os.FileInfo's methods.
type fileInfo struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	IsDir   bool        `json:"isDir"`
	Sys     interface{} `json:"sys"`
}

func newFileInfo(finfo os.FileInfo) fileInfo {
	return fileInfo{
		Name:    finfo.Name(),
		Size:    finfo.Size(),
		Mode:    finfo.Mode(),
		ModTime: finfo.ModTime(),
		IsDir:   finfo.IsDir(),
		Sys:     finfo.Sys(),
	}
}

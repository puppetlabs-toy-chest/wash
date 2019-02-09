package plugin

import (
	"context"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/log"
)

var slow = false

// DefaultTimeout is a default timeout for prefetched data.
const DefaultTimeout = 10 * time.Second

// Init sets up plugin core configuration on startup.
func Init(_slow bool) {
	slow = _slow
}

// ==== Plugin registry (FS) ====
//
// Here we implement directory methods for FS so
// that FUSE can recognize it as a valid root directory
//

var _ fs.FS = (*FS)(nil)

// NewFS creates a new FS.
func NewFS(plugins map[string]Entry) *FS {
	return &FS{Plugins: plugins, Entry: EntryT{"/"}}
}

// Root presents the root of the filesystem.
func (f *FS) Root() (fs.Node, error) {
	log.Printf("Entering root of filesystem")
	return &dir{f, ""}, nil
}

// LS lists all clients as directories.
func (f *FS) LS(_ context.Context) ([]Entry, error) {
	keys := make([]Entry, 0, len(f.Plugins))
	for _, v := range f.Plugins {
		keys = append(keys, v)
	}
	return keys, nil
}

var startTime = time.Now()

// Applies attributes where non-default, and sets defaults otherwise.
func applyAttr(a *fuse.Attr, attr *Attributes) {
	a.Valid = 1 * time.Second
	if attr.Valid != 0 {
		a.Valid = attr.Valid
	}

	// TODO: tie this to actual hard links in plugins
	a.Nlink = 1
	a.Mode = attr.Mode
	a.Size = attr.Size

	var zero time.Time
	a.Mtime = startTime
	if attr.Mtime != zero {
		a.Mtime = attr.Mtime
	}
	a.Atime = startTime
	if attr.Atime != zero {
		a.Atime = attr.Atime
	}
	a.Ctime = startTime
	if attr.Ctime != zero {
		a.Ctime = attr.Ctime
	}
	a.Crtime = startTime
}

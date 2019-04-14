// Package apifs is used by the Wash API to convert local files/directories
// into Wash entries
package apifs

import (
	"os"
	"syscall"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// fsnode => filesystem node
type fsnode struct {
	plugin.EntryBase
	path string
}

func newFSNode(finfo os.FileInfo, path string) *fsnode {
	// TODO: Need to case this on platform since finfo.Sys()
	// contains platform-specific data
	statT := finfo.Sys().(*syscall.Stat_t)
	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(timespecToTime(statT.Ctimespec)).
		SetAtime(timespecToTime(statT.Atimespec)).
		SetMtime(timespecToTime(statT.Mtimespec)).
		SetMode(os.FileMode(statT.Mode)).
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

func timespecToTime(t syscall.Timespec) time.Time {
	return time.Unix(t.Sec, t.Nsec)
}

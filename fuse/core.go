package fuse

import (
	"context"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

var startTime = time.Now()

// Root represents the root of the FUSE filesystem
type Root struct {
	Plugins map[string]plugin.Root
}

// Applies attributes where non-default, and sets defaults otherwise.
func applyAttr(a *fuse.Attr, attr *plugin.Attributes) {
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

func (f *Root) Name() string {
	return "/"
}

// Root presents the root of the filesystem.
func (f *Root) Root() (fs.Node, error) {
	log.Printf("Entering root of filesystem")
	return &dir{f, ""}, nil
}

// LS lists all clients as directories.
func (f *Root) LS(_ context.Context) ([]plugin.Entry, error) {
	keys := make([]plugin.Entry, 0, len(f.Plugins))
	for _, v := range f.Plugins {
		keys = append(keys, v)
	}
	return keys, nil
}

func ServeFuseFS(filesys *plugin.Registry, mountpoint string, debug bool) error {
	if debug {
		fuse.Debug = func(msg interface{}) {
			log.Debugf("%v", msg)
		}
	}

	log.Printf("Mounting at %v", mountpoint)
	fuseServer, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer fuseServer.Close()

	log.Warnf("Serving filesystem")
	if err := fs.Serve(fuseServer, &Root{Plugins: filesys.Plugins}); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-fuseServer.Ready
	if err := fuseServer.MountError; err != nil {
		return err
	}
	log.Warnf("Done")

	return nil
}

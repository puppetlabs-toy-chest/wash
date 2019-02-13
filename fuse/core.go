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

// ServeFuseFS serves the FUSE filesystem
func ServeFuseFS(filesys *plugin.Registry, mountpoint string, debug bool) (chan bool, error) {
	if debug {
		fuse.Debug = func(msg interface{}) {
			log.Debugf("%v", msg)
		}
	}

	log.Printf("FUSE: Mounting at %v", mountpoint)
	fuseConn, err := fuse.Mount(mountpoint)
	if err != nil {
		return nil, err
	}

	// Start the FUSE server
	fuseServerStoppedCh := make(chan struct{})
	go func() {
		defer close(fuseServerStoppedCh)
		defer func() {
			err := fuseConn.Close()
			if err != nil {
				log.Printf("FUSE: Error closing the connection: %v", err)
			}
		}()

		log.Printf("FUSE: Serving filesystem")
		if err := fs.Serve(fuseConn, &Root{Plugins: filesys.Plugins}); err != nil {
			log.Warnf("FUSE: fs.Serve errored with: %v", err)
		}

		// check if the mount process has an error to report
		<-fuseConn.Ready
		if err := fuseConn.MountError; err != nil {
			log.Warnf("FUSE: Mount process errored with: %v", err)
		}
		log.Printf("FUSE: Server was shut down")
	}()

	// Clean-up
	stopCh := make(chan bool)
	go func() {
		defer close(stopCh)
		<-stopCh

		log.Printf("FUSE: Shutting down the server")

		log.Printf("FUSE: Unmounting %v", mountpoint)
		if err := fuse.Unmount(mountpoint); err != nil {
			log.Warnf("FUSE: Failed to unmount %v: %v", mountpoint, err.Error())
			return
		}

		// Wait for the FUSE server to shutdown.
		<-fuseServerStoppedCh
	}()

	return stopCh, nil
}

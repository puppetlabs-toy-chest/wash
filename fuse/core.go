package fuse

import (
	"context"
	"os/user"
	"strconv"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

var startTime = time.Now()

// Root represents the root of the FUSE filesystem
type Root struct {
	Plugins map[string]plugin.Root
}

func getIDs() (uint32, uint32) {
	me, err := user.Current()
	if err != nil {
		log.Infof("Unable to fetch user: %v", err)
		return 0, 0
	}
	uid, err := strconv.ParseUint(me.Uid, 10, 32)
	if err != nil {
		log.Infof("Unable to parse uid: %v", err)
		return 0, 0
	}
	gid, err := strconv.ParseUint(me.Gid, 10, 32)
	if err != nil {
		log.Infof("Unable to parse gid: %v", err)
		return 0, 0
	}
	return uint32(uid), uint32(gid)
}

var uid, gid = getIDs()

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
	a.BlockSize = 4096
	a.Uid = uid
	a.Gid = gid
}

// Name returns '/', the name for the filesystem root.
func (f *Root) Name() string {
	return "/"
}

// Root presents the root of the filesystem.
func (f *Root) Root() (fs.Node, error) {
	log.Infof("Entering root of filesystem")
	return newDir(f, ""), nil
}

// LS lists all clients as directories.
func (f *Root) LS(_ context.Context) ([]plugin.Entry, error) {
	keys := make([]plugin.Entry, 0, len(f.Plugins))
	for _, v := range f.Plugins {
		keys = append(keys, v)
	}
	return keys, nil
}

// ServeFuseFS starts serving a fuse filesystem that lists the registered plugins.
func ServeFuseFS(filesys *plugin.Registry, mountpoint string) (chan bool, error) {
	fuse.Debug = func(msg interface{}) {
		log.Tracef("FUSE: %v", msg)
	}

	log.Infof("FUSE: Mounting at %v", mountpoint)
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
				log.Infof("FUSE: Error closing the connection: %v", err)
			}
		}()

		log.Infof("FUSE: Serving filesystem")
		if err := fs.Serve(fuseConn, &Root{Plugins: filesys.Plugins}); err != nil {
			log.Warnf("FUSE: fs.Serve errored with: %v", err)
		}

		// check if the mount process has an error to report
		<-fuseConn.Ready
		if err := fuseConn.MountError; err != nil {
			log.Warnf("FUSE: Mount process errored with: %v", err)
		}
		log.Infof("FUSE: Server was shut down")
	}()

	// Clean-up
	stopCh := make(chan bool)
	go func() {
		defer close(stopCh)
		<-stopCh

		log.Infof("FUSE: Shutting down the server")

		log.Infof("FUSE: Unmounting %v", mountpoint)
		if err := fuse.Unmount(mountpoint); err != nil {
			log.Warnf("FUSE: Failed to unmount %v: %v", mountpoint, err.Error())
			return
		}

		// Wait for the FUSE server to shutdown.
		<-fuseServerStoppedCh
	}()

	return stopCh, nil
}

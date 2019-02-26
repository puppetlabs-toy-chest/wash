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
	plugins []plugin.Entry
}

func newRoot(plugins map[string]plugin.Root) Root {
	root := Root{}

	root.plugins = make([]plugin.Entry, 0, len(plugins))
	for _, v := range plugins {
		root.plugins = append(root.plugins, v)
	}

	return root
}

// Name returns '/', the name for the filesystem root.
func (r *Root) Name() string {
	return "/"
}

// Root presents the root of the filesystem.
func (r *Root) Root() (fs.Node, error) {
	log.Infof("Entering root of filesystem")
	return newDir(r, ""), nil
}

// LS lists all clients as directories.
func (r *Root) LS(_ context.Context) ([]plugin.Entry, error) {
	return r.plugins, nil
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

type fuseNode interface {
	Entry() plugin.Entry
	String() string
}

func ftype(f fuseNode) string {
	if _, ok := f.(*dir); ok {
		return "d"
	} else {
		return "f"
	}
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

	a.Mtime = startTime
	if !attr.Mtime.IsZero() {
		a.Mtime = attr.Mtime
	}
	a.Atime = startTime
	if !attr.Atime.IsZero() {
		a.Atime = attr.Atime
	}
	a.Ctime = startTime
	if !attr.Ctime.IsZero() {
		a.Ctime = attr.Ctime
	}
	a.Crtime = startTime
	a.BlockSize = 4096
	a.Uid = uid
	a.Gid = gid
}

func attr(ctx context.Context, f fuseNode, a *fuse.Attr) error {
	attr := plugin.Attributes{}

	err := plugin.FillAttr(ctx, f.Entry(), f.String(), &attr)
	if _, ok := err.(plugin.ErrCouldNotDetermineSizeAttr); ok {
		log.Warnf("FUSE: Warn[Attr,%v]: %v", f, err)
	} else if err != nil {
		log.Warnf("FUSE: Error[Attr,%v]: %v", f, err)
		return err
	}

	applyAttr(a, &attr)
	log.Infof("FUSE: Attr[%v] %v %v", ftype(f), f, a)
	return nil
}

func listxattr(ctx context.Context, f fuseNode, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	log.Infof("FUSE: Listxattr[%v,pid=%v] %v", ftype(f), req.Pid, f)
	resp.Append("wash.id")
	return nil
}

func getxattr(ctx context.Context, f fuseNode, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	log.Infof("FUSE: Getxattr[%v,pid=%v] %v", ftype(f), req.Pid, f)
	switch req.Name {
	case "wash.id":
		resp.Xattr = []byte(f.String())
	}

	return nil
}

// ServeFuseFS starts serving a fuse filesystem that lists the registered plugins.
// It returns three values:
//   1. A channel to initiate the shutdown (stopCh).
//
//   2. A read-only channel that signals whether the server was shutdown
//
//   3. An error object
func ServeFuseFS(filesys *plugin.Registry, mountpoint string) (chan<- bool, <-chan struct{}, error) {
	fuse.Debug = func(msg interface{}) {
		log.Tracef("FUSE: %v", msg)
	}

	log.Infof("FUSE: Mounting at %v", mountpoint)
	fuseConn, err := fuse.Mount(mountpoint)
	if err != nil {
		return nil, nil, err
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

		root := newRoot(filesys.Plugins)
		if err := fs.Serve(fuseConn, &root); err != nil {
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

	return stopCh, fuseServerStoppedCh, nil
}

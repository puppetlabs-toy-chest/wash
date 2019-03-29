// Package fuse adapts wash plugin types to a FUSE filesystem.
package fuse

import (
	"context"
	"os"
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
	registry *plugin.Registry
}

func newRoot(registry *plugin.Registry) Root {
	return Root{registry: registry}
}

// Root presents the root of the filesystem.
func (r *Root) Root() (fs.Node, error) {
	log.Infof("Entering root of filesystem")
	return newDir(r.registry), nil
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

var attrRefreshInterval = 5 * time.Second

type fuseNode struct {
	ftype             string
	entry             plugin.Entry
	entryCreationTime time.Time
}

func newFuseNode(ftype string, entry plugin.Entry) *fuseNode {
	return &fuseNode{
		ftype:             ftype,
		entry:             entry,
		entryCreationTime: time.Now(),
	}
}

func (f *fuseNode) String() string {
	return plugin.Path(f.entry)
}

// Applies attributes where non-default, and sets defaults otherwise.
func (f *fuseNode) applyAttr(a *fuse.Attr, attr *plugin.EntryAttributes) {
	// This doesn't quite work for some reason.
	a.Valid = attrRefreshInterval

	// TODO: tie this to actual hard links in plugins
	a.Nlink = 1

	if attr.HasMode() {
		a.Mode = attr.Mode()
	} else if plugin.ListAction.IsSupportedOn(f.entry) {
		a.Mode = os.ModeDir | 0550
	} else {
		a.Mode = 0440
	}

	a.Size = ^uint64(0)
	if attr.HasSize() {
		a.Size = attr.Size()
	}

	a.Mtime = startTime
	if attr.HasMtime() {
		a.Mtime = attr.Mtime()
	}
	a.Atime = startTime
	if attr.HasAtime() {
		a.Atime = attr.Atime()
	}
	a.Ctime = startTime
	if attr.HasCtime() {
		a.Ctime = attr.Ctime()
	}
	a.Crtime = startTime
	a.BlockSize = 4096
	a.Uid = uid
	a.Gid = gid
}

func (f *fuseNode) Attr(ctx context.Context, a *fuse.Attr) error {
	// TODO: The code below's a temporary hack to show that the refreshing
	// behavior works. It blows up after several successive ls calls b/c each
	// child simultaneously attempts to refresh their attributes.
	if time.Since(f.entryCreationTime) >= attrRefreshInterval {
		err := plugin.RefreshAttributes(ctx, f.entry)
		if err != nil {
			log.Warnf("FUSE: Error[Attr,%v]: %v", f, err)
			return err
		}
	}
	attr := plugin.Attributes(f.entry)
	f.applyAttr(a, &attr)
	log.Infof("FUSE: Attr[%v] %v %v", f.ftype, f, a)
	return nil
}

func (f *fuseNode) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	log.Infof("FUSE: Listxattr[%v,pid=%v] %v", f.ftype, req.Pid, f)
	resp.Append("wash.id")
	return nil
}

func (f *fuseNode) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	log.Infof("FUSE: Getxattr[%v,pid=%v] %v", f.ftype, req.Pid, f)
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

		root := newRoot(filesys)
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
		if err = fuse.Unmount(mountpoint); err != nil {
			log.Warnf("FUSE: Shutdown failed: %v", err.Error())
			log.Warnf("FUSE: Manual cleanup required: umount %v", mountpoint)
		}
	}()

	return stopCh, fuseServerStoppedCh, nil
}

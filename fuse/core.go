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
	"github.com/puppetlabs/wash/activity"
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
	return newDir(nil, r.registry), nil
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

type fuseNode struct {
	ftype             string
	parent            *dir
	entry             plugin.Entry
	entryCreationTime time.Time
}

func newFuseNode(ftype string, parent *dir, entry plugin.Entry) *fuseNode {
	return &fuseNode{
		ftype:             ftype,
		parent:            parent,
		entry:             entry,
		entryCreationTime: time.Now(),
	}
}

func (f *fuseNode) String() string {
	return plugin.ID(f.entry)
}

// Applies attributes where non-default, and sets defaults otherwise.
func (f *fuseNode) applyAttr(a *fuse.Attr, attr *plugin.EntryAttributes, isdir bool) {
	// Setting a.Valid to 1 second avoids frequent Attr calls.
	a.Valid = 1 * time.Second

	// TODO: tie this to actual hard links in plugins
	a.Nlink = 1

	if attr.HasMode() {
		a.Mode = attr.Mode()
		// bazil/fuse appears to assume that character device implies device, and requires
		// device to be flagged to set char device.
		if a.Mode&os.ModeCharDevice == os.ModeCharDevice {
			a.Mode |= os.ModeDevice
		}
	} else if isdir {
		a.Mode = os.ModeDir | 0550
	} else {
		a.Mode = 0440
	}

	const blockSize = 4096
	if attr.HasSize() {
		a.Size = attr.Size()
	} else {
		// Default to the block size to encourage tools to read the file to determine its actual size.
		// We don't know the size, and cat at least ignores a file with size 0.
		a.Size = blockSize
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
	a.BlockSize = blockSize
	a.Uid = uid
	a.Gid = gid
}

// Re-discovers the source ancestor of the current node to get fresh data. It returns that ancestor
// and the path between it and the current node.
func (f *fuseNode) getSource() (plugin.Parent, []string) {
	cur, segments := f.parent, []string{plugin.CName(f.entry)}
	for cur != nil {
		if plugin.IsPrefetched(cur.entry) {
			segments = append([]string{plugin.CName(cur.entry)}, segments...)
			cur = cur.parent
		} else {
			// All dirs must contain a parent or they wouldn't have been able to create children.
			return cur.entry.(plugin.Parent), segments
		}
	}
	return nil, segments
}

// Re-discovers the current entry, based on any source ancestors.
func (f *fuseNode) refind(ctx context.Context) (plugin.Entry, error) {
	parent, segments := f.getSource()
	if parent == nil {
		return f.entry, nil
	}
	return plugin.FindEntry(ctx, parent, segments)
}

func (f *fuseNode) Attr(ctx context.Context, a *fuse.Attr) error {
	// Attr is not a particularly interesting call and happens a lot. Log it to debug like other
	// activity, but leave it out of activity because it introduces history entries for lots of
	// miscellaneous shell activity.
	log.Debugf("FUSE: Attr %v", f)

	// FUSE caches nodes for a long time, meaning there's a chance that
	// f's attributes are outdated. 'refind' requests the entry from its
	// parent to ensure it has updated attributes.
	updatedEntry, err := f.refind(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: Attr errored %v, %v", f, err)
		return err
	}
	attr := plugin.Attributes(updatedEntry)
	// NOTE: We could set f.entry to updatedEntry, but doing so would require
	// a separate mutex which may hinder performance. Since updating f.entry
	// is not strictly necessary for the other FUSE operations, we choose to
	// leave it alone.

	f.applyAttr(a, &attr, plugin.ListAction().IsSupportedOn(updatedEntry))
	log.Debugf("FUSE: Attr finished %v", f)
	return nil
}

// ServeFuseFS starts serving a fuse filesystem that lists the registered plugins.
// It returns three values:
//   1. A channel to initiate the shutdown (stopCh).
//
//   2. A read-only channel that signals whether the server was shutdown
//
//   3. An error object
func ServeFuseFS(filesys *plugin.Registry, mountpoint string) (chan<- context.Context, <-chan struct{}, error) {
	fuse.Debug = func(msg interface{}) {
		log.Tracef("FUSE: %v", msg)
	}

	log.Infof("FUSE: Mounting at %v", mountpoint)
	fuseConn, err := fuse.Mount(mountpoint)
	if err != nil {
		return nil, nil, err
	}

	// Start the FUSE server. We use the serverExitedCh to catch externally triggered unmounts.
	// If we're explicitly asked to shutdown the server, we want to wait until both Unmount and
	// Serve have exited before signaling completion.
	serverExitedCh := make(chan struct{})
	go func() {
		serverConfig := &fs.Config{
			WithContext: func(ctx context.Context, req fuse.Request) context.Context {
				pid := int(req.Hdr().Pid)
				return context.WithValue(ctx, activity.JournalKey, activity.JournalForPID(pid))
			},
		}
		server := fs.New(fuseConn, serverConfig)
		root := newRoot(filesys)
		if err := server.Serve(&root); err != nil {
			log.Warnf("FUSE: fs.Serve errored with: %v", err)
		}

		// check if the mount process has an error to report
		<-fuseConn.Ready
		if err := fuseConn.MountError; err != nil {
			log.Warnf("FUSE: Mount process errored with: %v", err)
		}
		log.Infof("FUSE: Serve complete")

		// Signal that Serve exited so the clean-up goroutine can close the stopped channel
		// if it hasn't already done so.
		defer close(serverExitedCh)
	}()

	// Clean-up
	stopCh := make(chan context.Context)
	stoppedCh := make(chan struct{})
	go func() {
		select {
		case <-stopCh:
			// Handle explicit shutdown
			log.Infof("FUSE: Shutting down the server")

			log.Infof("FUSE: Unmounting %v", mountpoint)
			if err = fuse.Unmount(mountpoint); err != nil {
				log.Warnf("FUSE: Shutdown failed: %v", err.Error())
				log.Warnf("FUSE: Manual cleanup required: umount %v", mountpoint)
			}
			log.Infof("FUSE: Unmount complete")
		case <-serverExitedCh:
			// Server exited on its own, fallthrough.
		}
		// Check that Serve has exited successfully in case we initiated the Unmount.
		<-serverExitedCh
		err := fuseConn.Close()
		if err != nil {
			log.Infof("FUSE: Error closing the connection: %v", err)
		}
		log.Infof("FUSE: Server shutdown complete")
		close(stoppedCh)
	}()

	return stopCh, stoppedCh, nil
}

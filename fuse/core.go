// Package fuse adapts wash plugin types to a FUSE filesystem.
package fuse

import (
	"context"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/analytics"
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
	ftype  string
	parent *dir
	entry  plugin.Entry
}

func newFuseNode(ftype string, parent *dir, entry plugin.Entry) fuseNode {
	return fuseNode{
		ftype:  ftype,
		parent: parent,
		entry:  entry,
	}
}

func (f *fuseNode) String() string {
	return plugin.ID(f.entry)
}

// Applies attributes where non-default, and sets defaults otherwise.
func applyAttr(a *fuse.Attr, attr plugin.EntryAttributes, defaultMode os.FileMode) {
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
	} else {
		a.Mode = defaultMode
	}

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
	if attr.HasCrtime() {
		a.Crtime = attr.Crtime()
	}
	a.BlockSize = 4096
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

// ServeFuseFS starts serving a fuse filesystem that lists the registered plugins.
// It returns three values:
//   1. A channel to initiate the shutdown (stopCh).
//
//   2. A read-only channel that signals whether the server was shutdown
//
//   3. An error object
func ServeFuseFS(
	filesys *plugin.Registry,
	mountpoint string,
	analyticsClient analytics.Client,
) (chan<- context.Context, <-chan struct{}, error) {
	fuse.Debug = func(msg interface{}) {
		log.Tracef("FUSE: %v", msg)
	}

	log.Infof("FUSE: Mounting at %v", mountpoint)
	fuseConn, err := fuse.Mount(mountpoint)
	if err != nil {
		return nil, nil, mountFailedErr(err)
	}

	// Start the FUSE server. We use the serverExitedCh to catch externally triggered unmounts.
	// If we're explicitly asked to shutdown the server, we want to wait until both Unmount and
	// Serve have exited before signaling completion.
	serverExitedCh := make(chan struct{})
	go func() {
		serverConfig := &fs.Config{
			WithContext: func(ctx context.Context, req fuse.Request) context.Context {
				pid := int(req.Hdr().Pid)
				newctx := context.WithValue(ctx, activity.JournalKey, activity.JournalForPID(pid))
				newctx = context.WithValue(newctx, analytics.ClientKey, analyticsClient)
				return newctx
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
				log.Warnf("FUSE: Shutdown failed: %v", err)
				log.Warnf("FUSE: Manual cleanup required: umount %v", mountpoint)

				// Retry in a loop until no longer blocked buy an open handle.
				// All errors are `*os.PathError`, so we just match a known error string.
				// Note that casing of the error message differs on macOS and Linux.
				for ; err != nil && strings.HasSuffix(strings.ToLower(err.Error()), "resource busy"); err = fuse.Unmount(mountpoint) {
					log.Debugf("FUSE: Unmount failed: %v", err)
					time.Sleep(3 * time.Second)
				}
				log.Debugf("FUSE: Unmount: %v", err)
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

// Package fuse adapts wash plugin types to a FUSE filesystem.
package fuse

import (
	"context"
	"fmt"
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
	parent            plugin.Parent
	entry             plugin.Entry
	entryCreationTime time.Time
}

func newFuseNode(ftype string, parent plugin.Parent, entry plugin.Entry) *fuseNode {
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
func (f *fuseNode) applyAttr(a *fuse.Attr, attr *plugin.EntryAttributes) {
	// Setting a.Valid to 1 second avoids frequent Attr calls.
	a.Valid = 1 * time.Second

	// TODO: tie this to actual hard links in plugins
	a.Nlink = 1

	if attr.HasMode() {
		a.Mode = attr.Mode()
	} else if plugin.ListAction().IsSupportedOn(f.entry) {
		a.Mode = os.ModeDir | 0550
	} else {
		a.Mode = 0440
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
	a.BlockSize = 4096
	a.Uid = uid
	a.Gid = gid
}

func (f *fuseNode) Attr(ctx context.Context, a *fuse.Attr) error {
	activity.Record(ctx, "FUSE: Attr %v", f)

	var attr plugin.EntryAttributes
	if f.parent == nil {
		attr = plugin.Attributes(f.entry)
	} else {
		// FUSE caches nodes for a long time, meaning there's a chance
		// that f's attributes are outdated. CachedList returns the entry's
		// and its sibling's updated attributes in a single request, so use
		// it to get f's updated attributes.
		entries, err := plugin.CachedList(ctx, f.parent)
		if err != nil {
			err := fmt.Errorf("could not refresh the attributes: %v", err)
			activity.Record(ctx, "FUSE: Attr errored %v, %v", f, err)
			return err
		}
		updatedEntry, ok := entries[plugin.CName(f.entry)]
		if !ok {
			err := fmt.Errorf("entry does not exist anymore")
			activity.Record(ctx, "FUSE: Attr errored %v, %v", f, err)
			return err
		}
		attr = plugin.Attributes(updatedEntry)
		// NOTE: We could set f.entry to updatedEntry, but doing so would require
		// a separate mutex which may hinder performance. Since updating f.entry
		// is not strictly necessary for the other FUSE operations, we choose to
		// leave it alone.
	}

	f.applyAttr(a, &attr)
	activity.Record(ctx, "FUSE: Attr finished %v", f)
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
		log.Infof("FUSE: Server was shut down")
	}()

	// Clean-up
	stopCh := make(chan context.Context)
	go func() {
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

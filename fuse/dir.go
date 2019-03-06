package fuse

import (
	"context"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE Directory Interface ====

type dir struct {
	entry plugin.Entry
	id    string
}

var _ fs.Node = (*dir)(nil)
var _ = fs.NodeRequestLookuper(&dir{})
var _ = fs.HandleReadDirAller(&dir{})

func newDir(e plugin.Entry, parent string) *dir {
	id := strings.TrimSuffix(parent, "/") + "/" + strings.TrimPrefix(e.Name(), "/")
	return &dir{e, id}
}

func (d *dir) Entry() plugin.Entry {
	return d.entry
}

func (d *dir) String() string {
	return d.id
}

// Attr returns the attributes of a directory.
func (d *dir) Attr(ctx context.Context, a *fuse.Attr) error {
	// TODO: need an enhancement to bazil.org/fuse to pass request to a method like Attr.
	return attr(context.WithValue(ctx, journal.Key, ""), d, a)
}

// Listxattr lists extended attributes for the resource.
func (d *dir) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	return listxattr(ctx, d, req, resp)
}

// Getxattr gets extended attributes for the resource.
func (d *dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return getxattr(ctx, d, req, resp)
}

func (d *dir) children(ctx context.Context) ([]plugin.Entry, error) {
	// Cache List requests. FUSE often lists the contents then immediately calls find on individual entries.
	if plugin.ListAction.IsSupportedOn(d.Entry()) {
		return plugin.CachedList(ctx, d.Entry().(plugin.Group), d.id)
	}

	return []plugin.Entry{}, fuse.ENOENT
}

// Lookup searches a directory for children.
func (d *dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	jid := journal.PIDToID(int(req.Pid))
	ctx = context.WithValue(ctx, journal.Key, jid)
	journal.Record(ctx, "FUSE: Find %v in %v", req.Name, d)

	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[Find,%v,%v]: %v", d, req.Name, err)
		journal.Record(ctx, "FUSE: Find %v in %v errored: %v", req.Name, d, err)
		return nil, fuse.ENOENT
	}

	for _, entry := range entries {
		if entry.Name() == req.Name {
			log.Infof("FUSE: Find[d,pid=%v] %v/%v", req.Pid, d, entry.Name())
			if plugin.ListAction.IsSupportedOn(entry) {
				childdir := newDir(entry, d.String())
				journal.Record(ctx, "FUSE: Found directory %v", childdir)
				// Prefetch directory entries into the cache
				go func() {
					_, err := childdir.children(context.WithValue(context.Background(), journal.Key, jid))
					journal.Record(ctx, "FUSE: Prefetching children of %v complete: %v", childdir, err)
				}()
				return childdir, nil
			}

			journal.Record(ctx, "FUSE: Found file %v/%v", d, entry.Name())
			return newFile(entry, d.String()), nil
		}
	}
	journal.Record(ctx, "FUSE: %v not found in %v", req.Name, d)
	return nil, fuse.ENOENT
}

// ReadDirAll lists all children of the directory.
func (d *dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// TODO: need an enhancement to bazil.org/fuse to pass request to a method like ReadDirAll.
	ctx = context.WithValue(ctx, journal.Key, "")
	journal.Record(ctx, "FUSE: List %v", d)

	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[List,%v]: %v", d, err)
		journal.Record(ctx, "FUSE: List %v errored: %v", d, err)
		return nil, err
	}

	log.Infof("FUSE: List %v in %v", len(entries), d)

	res := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		var de fuse.Dirent
		de.Name = entry.Name()
		if plugin.ListAction.IsSupportedOn(d.Entry()) {
			de.Type = fuse.DT_Dir
		}
		res[i] = de
	}
	journal.Record(ctx, "FUSE: Listed in %v: %+v", d, res)
	return res, nil
}

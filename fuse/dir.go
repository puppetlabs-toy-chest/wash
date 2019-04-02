package fuse

import (
	"context"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE Directory Interface ====

type dir struct {
	*fuseNode
}

var _ fs.Node = (*dir)(nil)
var _ = fs.NodeRequestLookuper(&dir{})
var _ = fs.HandleReadDirAller(&dir{})

func newDir(e plugin.Group) *dir {
	return &dir{newFuseNode("d", e)}
}

func (d *dir) children(ctx context.Context) ([]plugin.Entry, error) {
	// Cache List requests. FUSE often lists the contents then immediately calls find on individual entries.
	if plugin.ListAction.IsSupportedOn(d.entry) {
		return plugin.CachedList(ctx, d.entry.(plugin.Group))
	}

	return []plugin.Entry{}, fuse.ENOENT
}

// Lookup searches a directory for children.
func (d *dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	journal.Record(ctx, "FUSE: Find %v in %v", req.Name, d)

	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[Find,%v,%v]: %v", d, req.Name, err)
		journal.Record(ctx, "FUSE: Find %v in %v errored: %v", req.Name, d, err)
		return nil, fuse.ENOENT
	}

	for _, entry := range entries {
		cname := plugin.CName(entry)
		if cname == req.Name {
			log.Infof("FUSE: Find[d,pid=%v] %v/%v", req.Pid, d, cname)
			if plugin.ListAction.IsSupportedOn(entry) {
				childdir := newDir(entry.(plugin.Group))
				journal.Record(ctx, "FUSE: Found directory %v", childdir)
				// Prefetch directory entries into the cache
				go func() {
					// Need to use a different context here because we still want the prefetch
					// to happen even when the current context is cancelled.
					jid := journal.PIDToID(int(req.Pid))
					ctx := context.WithValue(context.Background(), journal.Key, jid)
					_, err := childdir.children(ctx)
					journal.Record(ctx, "FUSE: Prefetching children of %v complete: %v", childdir, err)
				}()
				return childdir, nil
			}

			journal.Record(ctx, "FUSE: Found file %v/%v", d, cname)
			return newFile(entry), nil
		}
	}
	journal.Record(ctx, "FUSE: %v not found in %v", req.Name, d)
	return nil, fuse.ENOENT
}

// ReadDirAll lists all children of the directory.
func (d *dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
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
		de.Name = plugin.CName(entry)
		if plugin.ListAction.IsSupportedOn(d.entry) {
			de.Type = fuse.DT_Dir
		}
		res[i] = de
	}
	journal.Record(ctx, "FUSE: Listed in %v: %+v", d, res)
	return res, nil
}

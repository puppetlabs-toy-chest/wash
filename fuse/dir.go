package fuse

import (
	"context"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/activity"
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

func newDir(p plugin.Group, e plugin.Group) *dir {
	return &dir{newFuseNode("d", p, e)}
}

func (d *dir) children(ctx context.Context) (map[string]plugin.Entry, error) {
	// Cache List requests. FUSE often lists the contents then immediately calls find on individual entries.
	if plugin.ListAction.IsSupportedOn(d.entry) {
		return plugin.CachedList(ctx, d.entry.(plugin.Group))
	}

	return map[string]plugin.Entry{}, fuse.ENOENT
}

// Lookup searches a directory for children.
func (d *dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	activity.Record(ctx, "FUSE: Find %v in %v", req.Name, d)

	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[Find,%v,%v,jid=%v]: %v", d, req.Name, activity.GetID(ctx), err)
		activity.Record(ctx, "FUSE: Find %v in %v errored: %v", req.Name, d, err)
		return nil, fuse.ENOENT
	}

	cname := req.Name
	entry, ok := entries[cname]
	if !ok {
		activity.Record(ctx, "FUSE: %v not found in %v", req.Name, d)
		return nil, fuse.ENOENT
	}

	log.Infof("FUSE: Find[d,jid=%v] %v/%v", activity.GetID(ctx), d, cname)
	if plugin.ListAction.IsSupportedOn(entry) {
		childdir := newDir(d.entry.(plugin.Group), entry.(plugin.Group))
		activity.Record(ctx, "FUSE: Found directory %v", childdir)
		return childdir, nil
	}

	activity.Record(ctx, "FUSE: Found file %v/%v", d, cname)
	return newFile(d.entry.(plugin.Group), entry), nil
}

// ReadDirAll lists all children of the directory.
func (d *dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	activity.Record(ctx, "FUSE: List %v", d)

	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[List,%v,jid=%v]: %v", d, activity.GetID(ctx), err)
		activity.Record(ctx, "FUSE: List %v errored: %v", d, err)
		return nil, err
	}

	log.Infof("FUSE: List[jid=%v] %v in %v", activity.GetID(ctx), len(entries), d)

	res := make([]fuse.Dirent, 0, len(entries))
	for cname, entry := range entries {
		var de fuse.Dirent
		de.Name = cname
		if plugin.ListAction.IsSupportedOn(entry) {
			de.Type = fuse.DT_Dir
		}
		res = append(res, de)
	}
	activity.Record(ctx, "FUSE: Listed in %v: %+v", d, res)
	return res, nil
}

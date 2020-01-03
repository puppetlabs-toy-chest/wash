package fuse

import (
	"context"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE Directory Interface ====

type dir struct {
	fuseNode
}

var _ fs.Node = (*dir)(nil)
var _ = fs.NodeRequestLookuper(&dir{})
var _ = fs.HandleReadDirAller(&dir{})

func newDir(p *dir, e plugin.Parent) *dir {
	return &dir{newFuseNode("d", p, e)}
}

func (d *dir) children(ctx context.Context) (*plugin.EntryMap, error) {
	// Check for an updated entry in case it has static state.
	updatedEntry, err := d.refind(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: List errored %v, %v", d, err)
		return nil, err
	}

	// Cache List requests. FUSE often lists the contents then immediately calls find on individual entries.
	if plugin.ListAction().IsSupportedOn(updatedEntry) {
		return plugin.ListWithAnalytics(ctx, updatedEntry.(plugin.Parent))
	}

	return nil, fuse.ENOENT
}

// Lookup searches a directory for children.
func (d *dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	// Find is only occasionally useful and happens a lot. Log it to debug like other activity, but
	// leave it out of activity because it introduces history entries for miscellaneous shell commands.
	log.Debugf("FUSE: Find %v in %v", req.Name, d)

	entries, err := d.children(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: Find %v in %v errored: %v", req.Name, d, err)
		return nil, fuse.ENOENT
	}

	cname := req.Name
	entry, ok := entries.Load(cname)
	if !ok {
		log.Debugf("FUSE: %v not found in %v", req.Name, d)
		return nil, fuse.ENOENT
	}

	if plugin.ListAction().IsSupportedOn(entry) {
		childdir := newDir(d, entry.(plugin.Parent))
		log.Debugf("FUSE: Found directory %v", childdir)
		return childdir, nil
	}

	log.Debugf("FUSE: Found file %v/%v", d, cname)
	return newFile(d, entry), nil
}

// ReadDirAll lists all children of the directory.
func (d *dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	activity.Record(ctx, "FUSE: List %v", d)

	entries, err := d.children(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: List %v errored: %v", d, err)
		return nil, err
	}

	res := make([]fuse.Dirent, 0, entries.Len())
	entries.Range(func(cname string, entry plugin.Entry) bool {
		var de fuse.Dirent
		de.Name = cname
		if plugin.ListAction().IsSupportedOn(entry) {
			de.Type = fuse.DT_Dir
		} else {
			de.Type = fuse.DT_File
		}
		res = append(res, de)
		return true
	})
	activity.Record(ctx, "FUSE: Listed in %v: %+v", d, res)
	return res, nil
}

func (d *dir) Attr(ctx context.Context, a *fuse.Attr) error {
	// FUSE caches nodes for a long time, meaning there's a chance that
	// f's attributes are outdated. 'refind' requests the entry from its
	// parent to ensure it has updated attributes.
	entry, err := d.refind(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: Attr errored %v, %v", d, err)
		return err
	}
	// NOTE: We could set f.entry to entry, but doing so would require
	// a separate mutex which may hinder performance. Since updating f.entry
	// is not strictly necessary for the other FUSE operations, we choose to
	// leave it alone.

	applyAttr(a, plugin.Attributes(entry), os.ModeDir|0550)
	// Attr is not a particularly interesting call and happens a lot. Log it to debug like other
	// activity, but leave it out of activity because it introduces history entries for lots of
	// miscellaneous shell activity.
	if d.parent == nil {
		log.Tracef("FUSE: Attr %v", d)
	} else {
		log.Debugf("FUSE: Attr %v: %+v", d, *a)
	}
	return nil
}

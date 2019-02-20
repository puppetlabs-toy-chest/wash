package fuse

import (
	"context"
	"os"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE Directory Interface ====

type dir struct {
	plugin.Entry
	id string
}

var _ fs.Node = (*dir)(nil)
var _ = fs.NodeRequestLookuper(&dir{})
var _ = fs.HandleReadDirAller(&dir{})

func newDir(e plugin.Entry, parent string) *dir {
	id := strings.TrimSuffix(parent, "/") + "/" + strings.TrimPrefix(e.Name(), "/")
	return &dir{e, id}
}

func (d *dir) String() string {
	return d.id
}

// Attr returns the attributes of a directory.
func (d *dir) Attr(ctx context.Context, a *fuse.Attr) error {
	var attr plugin.Attributes
	if file, ok := d.Entry.(plugin.File); ok {
		attr = file.Attr()
	}
	if attr.Mode == 0 {
		attr.Mode = os.ModeDir | 0550
	}
	applyAttr(a, &attr)
	log.Infof("FUSE: Attr[d] %v %v", d, a)
	return nil
}

// Listxattr lists extended attributes for the resource.
func (d *dir) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	log.Infof("FUSE: Listxattr[d,pid=%v] %v", req.Pid, d)
	resp.Append("wash.id")
	return nil
}

// Getxattr gets extended attributes for the resource.
func (d *dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	log.Infof("FUSE: Getxattr[d,pid=%v] %v", req.Pid, d)
	switch req.Name {
	case "wash.id":
		resp.Xattr = []byte(d.String())
	}

	return nil
}

func (d *dir) children(ctx context.Context) ([]plugin.Entry, error) {
	// Cache LS requests. FUSE often lists the contents then immediately calls find on individual entries.
	switch v := d.Entry.(type) {
	case plugin.Group:
		return plugin.CachedLS(v, d.id, ctx)
	default:
		return []plugin.Entry{}, fuse.ENOENT
	}
}

// Lookup searches a directory for children.
func (d *dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[Find,%v,%v]: %v", d, req.Name, err)
		return nil, err
	}

	for _, entry := range entries {
		if entry.Name() == req.Name {
			log.Infof("FUSE: Find[d,pid=%v] %v/%v", req.Pid, d.String(), entry.Name())
			switch v := entry.(type) {
			case plugin.Group:
				// Prefetch directory entries into the cache
				go func() { d.children(context.Background()) }()
				return newDir(v, d.String()), nil
			default:
				return newFile(v, d.String()), nil
			}
		}
	}
	return nil, fuse.ENOENT
}

// ReadDirAll lists all children of the directory.
func (d *dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := d.children(ctx)
	if err != nil {
		log.Warnf("FUSE: Error[List,%v]: %v", d, err)
		return nil, err
	}

	log.Infof("FUSE: List %v in %v", len(entries), d)

	res := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		var de fuse.Dirent
		de.Name = entry.Name()
		switch entry.(type) {
		case plugin.Group:
			de.Type = fuse.DT_Dir
		}
		res[i] = de
	}
	return res, nil
}

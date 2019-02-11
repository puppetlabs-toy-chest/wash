package fuse

import (
	"context"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
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
	return &dir{e, parent + "/" + e.Name()}
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
	log.Printf("Attr[d] %v %v", d, a)
	return nil
}

// Listxattr lists extended attributes for the resource.
func (d *dir) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	if resource, ok := d.Entry.(plugin.Resource); ok {
		_, err := resource.Metadata(ctx)
		if err != nil {
			log.Warnf("Error[Listxattr,%v]: %v", d, err)
			return err
		}
		// TODO: turn meta into a list of extended attributes
		//for k := range xattrs { resp.Append(k) }
	}
	log.Printf("Listxattr[d,pid=%v] %v", req.Pid, d)
	return nil
}

// Getxattr gets extended attributes for the resource.
func (d *dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	if req.Name == "com.apple.FinderInfo" {
		return nil
	}

	if resource, ok := d.Entry.(plugin.Resource); ok {
		_, err := resource.Metadata(ctx)
		if err != nil {
			log.Warnf("Error[Getxattr,%v,%v]: %v", d, req.Name, err)
			return err
		}
		// TODO: get specific attrbute from meta
		// resp.Xattr = xattrs[req.Name]
	}
	log.Printf("Getxattr[d,pid=%v] %v", req.Pid, d)
	return nil
}

func prefetch(entry plugin.Entry) {
	switch v := entry.(type) {
	case plugin.Group:
		go func() { v.LS(context.Background()) }()
	default:
		log.Debugf("Not sure how to prefetch for %v", v)
	}
}

func (d *dir) get(ctx context.Context) (entries []plugin.Entry, err error) {
	switch v := d.Entry.(type) {
	case plugin.Group:
		entries, err = v.LS(ctx)
	default:
		err = fuse.ENOENT
	}
	return entries, err
}

// Lookup searches a directory for children.
func (d *dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	entries, err := d.get(ctx)
	if err != nil {
		log.Warnf("Error[Find,%v,%v]: %v", d, req.Name, err)
		return nil, err
	}

	for _, entry := range entries {
		if entry.Name() == req.Name {
			log.Printf("Find[d,pid=%v] %v/%v", req.Pid, d.String(), entry.Name())
			prefetch(entry)
			switch v := entry.(type) {
			case plugin.Group, dir:
				log.Printf("New directory: %v", v)
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
	entries, err := d.get(ctx)
	if err != nil {
		log.Warnf("Error[List,%v]: %v", d, err)
		return nil, err
	}

	log.Printf("List %v in %v", len(entries), d)

	res := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		var de fuse.Dirent
		switch v := entry.(type) {
		case plugin.Group, dir:
			de.Name = v.Name()
			de.Type = fuse.DT_Dir
		default:
			de.Name = v.Name()
		}
		res[i] = de
	}
	return res, nil
}

package fuse

import (
	"context"
	"io"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

// ==== FUSE file Interface ====

type file struct {
	plugin.Entry
	parent string
}

var _ fs.Node = (*file)(nil)
var _ = fs.NodeOpener(&file{})
var _ = fs.NodeGetxattrer(&file{})
var _ = fs.NodeListxattrer(&file{})

func newFile(e plugin.Entry, parent string) *file {
	return &file{e, parent + "/" + e.Name()}
}

func (f *file) String() string {
	return f.parent + "/" + f.Name()
}

// Attr returns the attributes of a file.
func (f *file) Attr(ctx context.Context, a *fuse.Attr) error {
	var attr plugin.Attributes
	if file, ok := f.Entry.(plugin.File); ok {
		attr = file.Attr()
	} else if readable, ok := f.Entry.(plugin.Readable); ok {
		attr.Size = readable.Size()
	}
	if attr.Mode == 0 {
		attr.Mode = 0440
	}
	applyAttr(a, &attr)
	log.Printf("Attr[f] %v %v", f, a)
	return nil
}

// Listxattr lists extended attributes for the resource.
func (f *file) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	if resource, ok := f.Entry.(plugin.Resource); ok {
		_, err := resource.Metadata(ctx)
		if err != nil {
			log.Warnf("Error[Listxattr,%v]: %v", f, err)
			return err
		}
		// TODO: turn meta into a list of extended attributes
		//for k := range xattrs { resp.Append(k) }
	}
	log.Printf("Listxattr[f,pid=%v] %v", req.Pid, f)
	return nil
}

// Getxattr gets extended attributes for the resource.
func (f *file) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	if req.Name == "com.apple.FinderInfo" {
		return nil
	}

	if resource, ok := f.Entry.(plugin.Resource); ok {
		_, err := resource.Metadata(ctx)
		if err != nil {
			log.Warnf("Error[Getxattr,%v,%v]: %v", f, req.Name, err)
			return err
		}
		// TODO: get specific attrbute from meta
		// resp.Xattr = xattrs[req.Name]
	}
	log.Printf("Getxattr[f,pid=%v] %v", req.Pid, f)
	return nil
}

// Open a file for reading.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// Initiate content request and return a channel providing the results.
	log.Printf("Opening[pid=%v] %v", req.Pid, f)
	if rdr, ok := f.Entry.(plugin.Readable); ok {
		r, err := rdr.Open(ctx)
		if err != nil {
			log.Warnf("Error[Open,%v]: %v", f, err)
			return nil, err
		}
		log.Printf("Opened[pid=%v] %v", req.Pid, f)
		return &fileHandle{r: r, id: f.String()}, nil
	}
	log.Warnf("Error[Open,%v]: cannot open this entry", f)
	return nil, fuse.ENOTSUP
}

type fileHandle struct {
	r  io.ReaderAt
	id string
}

var _ fs.Handle = (*fileHandle)(nil)
var _ = fs.HandleReleaser(fileHandle{})
var _ = fs.HandleReader(fileHandle{})

// Release closes the open file.
func (fh fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Printf("Release[pid=%v] %v", req.Pid, fh.id)
	if closer, ok := fh.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Read fills a buffer with the requested amount of data from the file.
func (fh fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	buf := make([]byte, req.Size)
	n, err := fh.r.ReadAt(buf, req.Offset)
	if err == io.EOF {
		err = nil
	}
	log.Printf("Read[pid=%v] %v, %v/%v bytes starting at %v: %v", fh.id, req.Pid, n, req.Size, req.Offset, err)
	resp.Data = buf[:n]
	return err
}

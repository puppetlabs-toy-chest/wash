package fuse

import (
	"context"
	"fmt"
	"io"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE file Interface ====

type file struct {
	plugin.Entry
	id      string
	content datastore.Var
}

var _ fs.Node = (*file)(nil)
var _ = fs.NodeOpener(&file{})
var _ = fs.NodeGetxattrer(&file{})
var _ = fs.NodeListxattrer(&file{})

func newFile(e plugin.Entry, parent string) *file {
	return &file{e, parent + "/" + e.Name(), datastore.NewVar(5 * time.Second)}
}

func (f *file) String() string {
	return f.id
}

// Attr returns the attributes of a file.
func (f *file) Attr(ctx context.Context, a *fuse.Attr) error {
	attr := plugin.Attributes{Size: plugin.SizeUnknown}
	if item, ok := f.Entry.(plugin.File); ok {
		attr = item.Attr()
	}

	if item, ok := f.Entry.(plugin.Readable); attr.Size == plugin.SizeUnknown && ok {
		raw, err := f.content.Update(func() (interface{}, error) {
			log.Infof("FUSE: [Attr,%v]: Recomputing the file's size attr", f)
			return item.Open(ctx)
		})
		if err != nil {
			log.Warnf("FUSE: Error[Attr,%v]: %v", f, err)
			return err
		}
		content := raw.(plugin.SizedReader)

		size := content.Size()
		if size < 0 {
			err := fmt.Errorf("Returned a negative value for the size: %v", size)
			log.Warnf("FUSE: Error[Attr,%v]: %v", f, err)
			return err
		}

		attr.Size = uint64(size)
	}

	if attr.Mode == 0 {
		attr.Mode = 0440
	}
	applyAttr(a, &attr)
	log.Infof("FUSE: Attr[f] %v %v", f, a)
	return nil
}

// Listxattr lists extended attributes for the resource.
func (f *file) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	log.Infof("FUSE: Listxattr[f,pid=%v] %v", req.Pid, f)
	resp.Append("wash.id")
	return nil
}

// Getxattr gets extended attributes for the resource.
func (f *file) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	log.Infof("FUSE: Getxattr[f,pid=%v] %v", req.Pid, f)
	switch req.Name {
	case "wash.id":
		resp.Xattr = []byte(f.String())
	}

	return nil
}

// Open a file for reading.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// Initiate content request and return a channel providing the results.
	log.Infof("FUSE: Opening[pid=%v] %v", req.Pid, f)
	if readable, ok := f.Entry.(plugin.Readable); ok {
		raw, err := f.content.Update(func() (interface{}, error) {
			log.Infof("FUSE: [Open,%v]: Recomputing the file contents", f)
			return readable.Open(ctx)
		})
		if err != nil {
			log.Warnf("FUSE: Error[Open,%v]: %v", f, err)
			return nil, err
		}
		content := raw.(plugin.SizedReader)

		log.Infof("FUSE: Opened[pid=%v] %v", req.Pid, f)
		return &fileHandle{r: content, id: f.String()}, nil
	}
	log.Warnf("FUSE: Error[Open,%v]: cannot open this entry", f)
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
	log.Infof("FUSE: Release[pid=%v] %v", req.Pid, fh.id)
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
	log.Infof("FUSE: Read[pid=%v] %v, %v/%v bytes starting at %v: %v", fh.id, req.Pid, n, req.Size, req.Offset, err)
	resp.Data = buf[:n]
	return err
}

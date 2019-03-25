package fuse

import (
	"context"
	"io"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

// ==== FUSE file Interface ====

type file struct {
	entry plugin.Entry
}

var _ fs.Node = (*file)(nil)
var _ = fs.NodeOpener(&file{})
var _ = fs.NodeGetxattrer(&file{})
var _ = fs.NodeListxattrer(&file{})

func newFile(e plugin.Entry) *file {
	return &file{e}
}

func (f *file) Entry() plugin.Entry {
	return f.entry
}

func (f *file) String() string {
	return plugin.Path(f.entry)
}

// Attr returns the attributes of a file.
func (f *file) Attr(ctx context.Context, a *fuse.Attr) error {
	// TODO: need an enhancement to bazil.org/fuse to pass request to a method like Attr.
	return attr(context.WithValue(ctx, journal.Key, ""), f, a)
}

// Listxattr lists extended attributes for the resource.
func (f *file) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	return listxattr(ctx, f, req, resp)
}

// Getxattr gets extended attributes for the resource.
func (f *file) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return getxattr(ctx, f, req, resp)
}

// Open a file for reading.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	ctx = context.WithValue(ctx, journal.Key, journal.PIDToID(int(req.Pid)))
	journal.Record(ctx, "FUSE: Open %v", f)

	// Initiate content request and return a channel providing the results.
	log.Infof("FUSE: Opening[pid=%v] %v", req.Pid, f)
	if plugin.ReadAction.IsSupportedOn(f.Entry()) {
		content, err := plugin.CachedOpen(ctx, f.Entry().(plugin.Readable))
		if err != nil {
			log.Warnf("FUSE: Error[Open,%v]: %v", f, err)
			journal.Record(ctx, "FUSE: Open %v errored: %v", f, err)
			return nil, err
		}

		log.Infof("FUSE: Opened[pid=%v] %v", req.Pid, f)
		journal.Record(ctx, "FUSE: Opened %v", f)
		return &fileHandle{r: content, id: f.String()}, nil
	}
	log.Warnf("FUSE: Error[Open,%v]: cannot open this entry", f)
	journal.Record(ctx, "FUSE: Open unsupported on %v", f)
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
	ctx = context.WithValue(ctx, journal.Key, journal.PIDToID(int(req.Pid)))

	log.Infof("FUSE: Release[pid=%v] %v", req.Pid, fh.id)
	journal.Record(ctx, "FUSE: Release %v", fh.id)
	if closer, ok := fh.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Read fills a buffer with the requested amount of data from the file.
func (fh fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	ctx = context.WithValue(ctx, journal.Key, journal.PIDToID(int(req.Pid)))

	buf := make([]byte, req.Size)
	n, err := fh.r.ReadAt(buf, req.Offset)
	if err == io.EOF {
		err = nil
	}
	log.Infof("FUSE: Read[pid=%v] %v, %v/%v bytes starting at %v: %v", fh.id, req.Pid, n, req.Size, req.Offset, err)
	journal.Record(ctx, "FUSE: Read %v/%v bytes starting at %v from %v: %v", n, req.Size, req.Offset, fh.id, err)
	resp.Data = buf[:n]
	return err
}

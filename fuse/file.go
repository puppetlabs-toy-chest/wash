package fuse

import (
	"context"
	"io"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// ==== FUSE file Interface ====

type file struct {
	*fuseNode
}

var _ fs.Node = (*file)(nil)
var _ = fs.NodeOpener(&file{})

func newFile(p *dir, e plugin.Entry) *file {
	return &file{newFuseNode("f", p, e)}
}

// Open a file for reading.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	activity.Record(ctx, "FUSE: Open %v", f)

	// Check for an updated entry in case it has static state.
	updatedEntry, err := f.refind(ctx)
	if err != nil {
		activity.Record(ctx, "FUSE: Open errored %v, %v", f, err)
		return nil, err
	}

	// Initiate content request and return a channel providing the results.
	if plugin.ReadAction().IsSupportedOn(updatedEntry) {
		content, err := plugin.CachedOpen(ctx, updatedEntry.(plugin.Readable))
		if err != nil {
			activity.Record(ctx, "FUSE: Open %v errored: %v", f, err)
			return nil, err
		}

		activity.Record(ctx, "FUSE: Opened %v", f)
		return &fileHandle{r: content, id: f.String()}, nil
	}
	activity.Record(ctx, "FUSE: Open unsupported on %v", f)
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
	activity.Record(ctx, "FUSE: Release %v", fh.id)
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
	activity.Record(ctx, "FUSE: Read %v/%v bytes starting at %v from %v: %v", n, req.Size, req.Offset, fh.id, err)
	resp.Data = buf[:n]
	return err
}

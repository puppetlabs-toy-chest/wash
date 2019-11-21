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

// Open a file for reading or writing.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	activity.Record(ctx, "FUSE: Open %v", f)

	// Check for an updated entry in case it has static state.
	updatedEntry, err := f.refind(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: Open errored %v, %v", f, err)
		return nil, err
	}

	var fh fileHandle
	fh.id = f.String()

	// Initiate content request and return a channel providing the results.
	if plugin.ReadAction().IsSupportedOn(updatedEntry) {
		content, err := plugin.OpenWithAnalytics(ctx, updatedEntry.(plugin.Readable))
		if err != nil {
			activity.Warnf(ctx, "FUSE: Open %v errored: %v", f, err)
			return nil, err
		}

		activity.Record(ctx, "FUSE: Opened %v", f)
		fh.r = content
	}

	if plugin.WriteAction().IsSupportedOn(updatedEntry) {
		fh.w = updatedEntry.(plugin.Writable)
	}

	if fh.r != nil || fh.w != nil {
		return &fh, nil
	}

	activity.Record(ctx, "FUSE: Open unsupported on %v", f)
	return nil, fuse.ENOTSUP
}

type fileHandle struct {
	r  io.ReaderAt
	w  plugin.Writable
	id string
}

var _ fs.Handle = (*fileHandle)(nil)
var _ = fs.HandleReleaser(fileHandle{})
var _ = fs.HandleReader(fileHandle{})
var _ = fs.HandleWriter(fileHandle{})

func (fh fileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	activity.Record(ctx, "FUSE: Release %v", fh.id)
	if closer, ok := fh.r.(io.Closer); ok {
		return closer.Close()
	}

	if closer, ok := fh.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

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

func (fh fileHandle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	n, err := fh.w.Write(ctx, req.Offset, req.Data)
	resp.Size = n
	activity.Record(ctx, "FUSE: Write %v/%v bytes starting at %v from %v: %v", n, len(req.Data), req.Offset, fh.id, err)
	return err
}

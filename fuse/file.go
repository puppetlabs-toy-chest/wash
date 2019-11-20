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

	// Prepare a reader for reading content.
	if plugin.ReadAction().IsSupportedOn(updatedEntry) {
		fh.r = updatedEntry.(plugin.Readable)

		attrs := plugin.Attributes(updatedEntry)
		if !attrs.HasSize() {
			// Request data to set the Size attribute.
			if _, err := fh.r.Read(ctx, nil, 0); err != nil {
				activity.Warnf(ctx, "FUSE: Open failed on %v attempting to prefill content: %w", f, err)
				return nil, err
			}
		}
	}

	// Prepare a writer for making updates.
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
	r  plugin.Readable
	w  plugin.Writable
	id string
}

var _ fs.Handle = (*fileHandle)(nil)
var _ = fs.HandleReader(fileHandle{})
var _ = fs.HandleWriter(fileHandle{})

func (fh fileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if fh.r == nil {
		panic("Should not have permission to read something that's not Readable")
	}
	buf := make([]byte, req.Size)
	n, err := plugin.ReadWithAnalytics(ctx, fh.r, buf, req.Offset)
	if err == io.EOF {
		err = nil
	}
	activity.Record(ctx, "FUSE: Read %v/%v bytes starting at %v from %v: %v", n, req.Size, req.Offset, fh.id, err)
	resp.Data = buf[:n]
	return err
}

func (fh fileHandle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if fh.w == nil {
		panic("Should not have permission to write something that's not Writable")
	}
	n, err := plugin.WriteWithAnalytics(ctx, fh.w, req.Data, req.Offset)
	resp.Size = n
	activity.Record(ctx, "FUSE: Wrote %v/%v bytes starting at %v from %v: %v", n, len(req.Data), req.Offset, fh.id, err)
	return err
}

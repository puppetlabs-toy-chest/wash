package fuse

import (
	"context"
	"io"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"bazil.org/fuse/fuseutil"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// ==== FUSE file Interface ====

type file struct {
	fuseNode

	mux sync.Mutex
	// Handles with in-progress writes
	writers map[fuse.HandleID]struct{}
	// Only valid if len(writers) > 0
	data []byte
	// Size for content during writing
	size uint64
}

func newFile(p *dir, e plugin.Entry) *file {
	return &file{fuseNode: newFuseNode("f", p, e), writers: make(map[fuse.HandleID]struct{})}
}

var _ = fs.Node(&file{})
var _ = fs.Handle(&file{})

func (f *file) Attr(ctx context.Context, a *fuse.Attr) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	if len(f.writers) == 0 {
		// Fetch updated attributes.
		entry, err := f.refind(ctx)
		if err != nil {
			activity.Warnf(ctx, "FUSE: Attr errored %v, %v", f, err)
			return err
		}
		f.entry = entry
	}

	f.fillAttr(a)
	activity.Record(ctx, "FUSE: Attr %v: %+v", f, *a)
	return nil
}

func (f *file) fillAttr(a *fuse.Attr) {
	attr := plugin.Attributes(f.entry)
	applyAttr(a, attr, getFileMode(f.entry))

	if len(f.writers) != 0 {
		// Use whatever size we know locally.
		a.Size = f.size
	}
}

func getFileMode(entry plugin.Entry) os.FileMode {
	var mode os.FileMode
	if plugin.WriteAction().IsSupportedOn(entry) {
		mode |= 0220
	}
	if plugin.ReadAction().IsSupportedOn(entry) ||
		plugin.StreamAction().IsSupportedOn(entry) {
		mode |= 0440
	}
	return mode
}

var _ = fs.NodeOpener(&file{})

// Open a file for reading or writing.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	activity.Record(ctx, "FUSE: Open %v: %+v", f, *req)

	// Check for an updated entry in case it has static state.
	entry, err := f.refind(ctx)
	if err != nil {
		activity.Warnf(ctx, "FUSE: Open errored %v, %v", f, err)
		return nil, err
	}
	f.entry = entry

	readable := plugin.ReadAction().IsSupportedOn(f.entry)
	writable := plugin.WriteAction().IsSupportedOn(f.entry)
	switch {
	case req.Flags.IsReadOnly() && !readable:
		activity.Record(ctx, "FUSE: Open read-only unsupported on %v", f)
		return nil, fuse.ENOTSUP
	case req.Flags.IsWriteOnly() && !writable:
		activity.Record(ctx, "FUSE: Open write-only unsupported on %v", f)
		return nil, fuse.ENOTSUP
	case req.Flags.IsReadWrite() && (!readable || !writable):
		activity.Record(ctx, "FUSE: Open read-write unsupported on %v", f)
		return nil, fuse.ENOTSUP
	}

	if req.Flags.IsReadOnly() || req.Flags.IsReadWrite() {
		attr := plugin.Attributes(entry)
		if !attr.HasSize() {
			// The entry's content size is unknown so open the file in direct IO mode. This enables FUSE
			// to still read the entry's content so that built-in tools like cat and grep still work.
			resp.Flags |= fuse.OpenDirectIO
		}
	}
	return f, nil
}

var _ = fs.HandleReleaser(&file{})

func (f *file) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if req.ReleaseFlags&fuse.ReleaseFlush != 0 {
		err := f.Flush(ctx, &fuse.FlushRequest{
			Header:    req.Header,
			Handle:    req.Handle,
			LockOwner: uint64(req.LockOwner),
		})
		if err != nil {
			return err
		}
	}

	activity.Record(ctx, "FUSE: Release %v: %+v", f, *req)
	return nil
}

var _ = fs.HandleReader(&file{})

func (f *file) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	if len(f.writers) == 0 {
		data, err := plugin.ReadWithAnalytics(ctx, f.entry, int64(req.Size), req.Offset)
		if err != nil && err != io.EOF {
			// If we don't ignore EOF, then cat will display an input/output error message
			// for entries with unknown content size.
			return err
		}
		resp.Data = data
	} else {
		fuseutil.HandleRead(req, resp, f.data)
	}

	activity.Record(ctx, "FUSE: Read %v/%v bytes starting at %v from %v", len(resp.Data), req.Size, req.Offset, f)
	return nil
}

var _ = fs.HandleWriter(&file{})

func (f *file) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	// Ensure handle is in list of writers.
	f.writers[req.Handle] = struct{}{}

	// If starting write beyond the current length, read to fill it in.
	if start := int64(len(f.data)); req.Offset > start {
		data, err := f.load(ctx, start, req.Offset)
		if err != nil {
			return err
		}
		f.data = append(f.data, data...)
	}

	// Expand the buffer if necessary and update known size.
	newLen := req.Offset + int64(len(req.Data))
	if newLen := int(newLen); newLen > len(f.data) {
		f.data = append(f.data, make([]byte, newLen-len(f.data))...)
	}
	// Write-only entries are assumed not to have a size.
	if plugin.ReadAction().IsSupportedOn(f.entry) && f.size < uint64(newLen) {
		f.size = uint64(newLen)
	}

	resp.Size = copy(f.data[req.Offset:], req.Data)
	activity.Record(ctx, "FUSE: Write %v/%v bytes starting at %v from %v", resp.Size, len(req.Data), req.Offset, f)
	return nil
}

func (f *file) load(ctx context.Context, start, end int64) ([]byte, error) {
	if !plugin.ReadAction().IsSupportedOn(f.entry) {
		activity.Record(ctx, "FUSE: Non-contiguous writes (at %v) unsupported on %v", start, f)
		return nil, fuse.ENOTSUP
	}

	return plugin.Read(ctx, f.entry, end-start, start)
}

var _ = fs.HandleFlusher(&file{})

// Note that this implementation of Flush only calls plugin.Write if there were previous calls to
// Write or Setattr. It doesn't check whether the data that's there matches what we're writing.
func (f *file) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	f.mux.Lock()
	defer f.mux.Unlock()
	activity.Record(ctx, "FUSE: Flush %v: %+v", f, *req)
	var releasedWriter bool

	// If this handle had an open writer, write current data.
	if _, ok := f.writers[req.Handle]; ok {
		dataLen := int64(len(f.data))
		// Write-only entries don't track size.
		if plugin.ReadAction().IsSupportedOn(f.entry) && uint64(dataLen) > f.size {
			panic("Size was not kept up-to-date with changes to data.")
		}

		if uint64(dataLen) < f.size {
			// Missing some data, load the remainder before writing.
			data, err := f.load(ctx, dataLen, int64(f.size))
			if err != nil {
				return err
			}
			f.data = append(f.data, data...)
		}

		if err := plugin.WriteWithAnalytics(ctx, f.entry.(plugin.Writable), f.data); err != nil {
			return err
		}
		delete(f.writers, req.Handle)
		releasedWriter = true
	}

	if len(f.writers) == 0 && releasedWriter {
		// Ensure data is released. Leave size so we can use it for future attribute requests.
		f.data = nil

		// Invalidate cache on the entry and its parent. Need to get updated content and size.
		plugin.ClearCacheFor(plugin.ID(f.entry), true)
	}
	return nil
}

var _ = fs.NodeSetattrer(&file{})

func (f *file) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	f.mux.Lock()
	defer f.mux.Unlock()
	activity.Record(ctx, "FUSE: Setattr[%v] %v: %+v", req.Handle, f, *req)

	if req.Valid.Size() {
		if req.Valid.Handle() {
			// Ensure handle is in list of writers.
			f.writers[req.Handle] = struct{}{}
		} else {
			// No guarantee we'll ever write the change. If this is ever necessary, we could update it
			// to immediately do a plugin.Write.
			return fuse.ENOTSUP
		}

		// Update known size. If the caller tries to increase the size of a Write-only entry, Flush
		// will error because we won't be able to read to fill it in. We choose to error instead of
		// filling with null data because there's no obvious use-case for supporting it.
		if f.size != req.Size {
			f.size = req.Size
		}

		// Shrink data if too large. Filling if too small is left for Write/Flush to deal with.
		if uint64(len(f.data)) > f.size {
			f.data = f.data[:f.size]
		}
	}

	f.fillAttr(&resp.Attr)
	return nil
}

// Needs to be defined or vim gets an fuse.EIO error on Fsync.
var _ = fs.NodeFsyncer(&file{})

func (f *file) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	// As noted in the docs for fs.NodeFsyncer, this should be implemented on a Handle. Write Fsync
	// should be unnecessary because Flush handles complete serialization out. On a handle opened
	// for reading, we could potentially invalidate the Wash cache and re-request data from the
	// plugin, but in most cases that doesn't seem to be necessary.
	activity.Record(ctx, "FUSE: Fsync %v: %+v", f, *req)
	return fuse.ENOSYS
}

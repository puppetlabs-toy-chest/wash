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
	// Size for content during writing or with unspecified size attribute
	size uint64
	// Tracks whether we still need to query plugin.Size for the entry's size before using size
	sizeValid bool
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

	if len(f.writers) != 0 || !attr.HasSize() {
		// Use whatever size we know locally. Retrieving content can be expensive so we settle for
		// including size only when it's been retreived previously by opening the file.
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

// Open an entry for reading or writing. Several patterns exist for how to interact with entries.
// - An entry that only supports Read can only be opened ReadOnly.
// - An entry that supports both Read and Write can be opened in any mode.
// - An entry that only supports Write can only be opened WriteOnly.
//
// When writing and flushing a file, we may call Read on the entry (if it supports Read) even if
// opened WriteOnly. That only happens when performing a partial write, or when its Size attribute
// isn't set and there are no calls to Setattr to define the file size.
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
		activity.Warnf(ctx, "FUSE: Open read-only unsupported on %v", f)
		return nil, fuse.ENOTSUP
	case req.Flags.IsWriteOnly() && !writable:
		activity.Warnf(ctx, "FUSE: Open write-only unsupported on %v", f)
		return nil, fuse.ENOTSUP
	case req.Flags.IsReadWrite() && (!readable || !writable):
		activity.Warnf(ctx, "FUSE: Open read-write unsupported on %v", f)
		return nil, fuse.ENOTSUP
	}

	if req.Flags.IsReadOnly() || req.Flags.IsReadWrite() {
		if attr := plugin.Attributes(entry); !attr.HasSize() {
			// The entry's content size is unknown so open the file in direct IO mode. This enables FUSE
			// to still read the entry's content so that built-in tools like cat and grep still work.
			resp.Flags |= fuse.OpenDirectIO

			// Also set the size for editors that won't read anything on an apparently empty file. This
			// doesn't help with cat/grep because they check attributes before opening the file.
			f.size, err = plugin.Size(ctx, entry)
			if err != nil {
				return nil, err
			}
		} else {
			f.size = attr.Size()
		}

		f.sizeValid = true
	} else if !readable {
		// Reported size is irrelevant for write-only entries.
		f.sizeValid = true
	}

	return f, nil
}

var _ = fs.HandleReleaser(&file{})

func (f *file) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if req.ReleaseFlags&fuse.ReleaseFlush != 0 {
		activity.Record(ctx, "FUSE: Invoking Flush for Release on %v", f)
		err := f.Flush(ctx, &fuse.FlushRequest{
			Header:    req.Header,
			Handle:    req.Handle,
			LockOwner: uint64(req.LockOwner),
		})
		if err != nil {
			return err
		}
	}

	if _, ok := f.writers[req.Handle]; ok {
		delete(f.writers, req.Handle)

		if len(f.writers) == 0 {
			// If we just released the last writer, release the data buffer to conserve memory and
			// invalidate cache on the entry and its parent so we get updated content and size on the
			// next request. Leave size so we can use it for future attribute requests.
			f.data = nil
			plugin.ClearCacheFor(plugin.ID(f.entry), true)
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
	// Size is irrelevant for write-only entries.
	if plugin.ReadAction().IsSupportedOn(f.entry) && f.size < uint64(newLen) {
		f.size = uint64(newLen)
	}

	resp.Size = copy(f.data[req.Offset:], req.Data)
	activity.Record(ctx, "FUSE: Write %v/%v bytes starting at %v from %v", resp.Size, len(req.Data), req.Offset, f)
	return nil
}

func (f *file) load(ctx context.Context, start, end int64) ([]byte, error) {
	if !plugin.ReadAction().IsSupportedOn(f.entry) {
		activity.Warnf(ctx, "FUSE: Non-contiguous writes (at %v) unsupported on %v", start, f)
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

	// If this handle had an open writer, write current data.
	if _, ok := f.writers[req.Handle]; ok {
		dataLen := int64(len(f.data))
		// Write-only entries don't track size.
		if plugin.ReadAction().IsSupportedOn(f.entry) && uint64(dataLen) > f.size {
			panic("Size was not kept up-to-date with changes to data.")
		}

		if !f.sizeValid {
			// If this is a readable entry that's opened WriteOnly and hasn't been truncated, we haven't
			// determined its size. Get it now so we can buffer as needed.
			size, err := plugin.Size(ctx, f.entry)
			if err != nil {
				return err
			}
			f.size = size
			f.sizeValid = true
		}

		if uint64(dataLen) < f.size {
			// Missing some data, load the remainder before writing.
			data, err := f.load(ctx, dataLen, int64(f.size))
			if err != nil && err != io.EOF {
				return err
			}
			f.data = append(f.data, data...)

			// If still too small then something increased the size beyond the original.
			// Fill with null characters.
			if sz := uint64(len(f.data)); sz < f.size {
				f.data = append(f.data, make([]byte, f.size-sz)...)
			}
		}

		if err := plugin.WriteWithAnalytics(ctx, f.entry.(plugin.Writable), f.data); err != nil {
			activity.Warnf(ctx, "FUSE: Error writing %v: %v", f, err)
			return err
		}
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
		f.size = req.Size
		f.sizeValid = true
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

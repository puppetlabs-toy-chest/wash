package fuse

import (
	"context"
	"io"
	"os"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"bazil.org/fuse/fuseutil"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// ==== FUSE file Interface ====

// file's implementation differs primarily on two axes: whether `Size` is set, and whether we're
// currently writing.
// - a *file-like* entry declares a `Size` in its `Attributes`
//   - reads and writes are symmetric; the kernel page cache will be used, and the file size
//     represents the current local content size of the file
//   - when writing, `data` represents the local content of the file; changes to its size will
//     usually be reflected in `data` and reads will be served from it
//   - the file's size will be `readSize`; `data` will be resized as-needed to delay loading data
//     from `plugin.Read` until it's needed (either a `Flush` or non-contiguous `Write`) and
//     differences between `readSize` and `len(data)` will be resolved when `Flush` is called to
//     avoid unnecessary calls to `plugin.Read` when overwriting a file
//   - when not writing, data is read directly from `plugin.Read`
// - a *non-file-like* entry has `Size` unset
//   - read always pulls from `plugin.Read` and writes are buffered independently
//   - `data` stores only the data to be written, and is not initialized from `plugin.Read`
//   - the file's size will be reported as its readable size; it will not reflect calls to `write`
//
// Note that writes only result in calling `plugin.Write` when a file handle is closed by the OS
// (triggering a call to `Flush` as noted in https://libfuse.github.io/doxygen/structfuse__operations.html#ad4ec9c309072a92dd82ddb20efa4ab14)
// Writing with multiple handles will be protected by `mux`, but all writes will operate on the
// same `data` and the first handle close will trigger `plugin.Write`.
//
// `readSize` will always be initialized from either the `Size` attribute, or if unset then the
// length of data available to read.
//
// `writers` are used to track in-progress writes so we know when to `plugin.Write` on `Flush`
type file struct {
	fuseNode

	mux sync.Mutex
	// Handles with in-progress writes
	writers map[fuse.HandleID]struct{}
	// Only valid if len(writers) > 0
	data []byte
	// Size of readable content, necessary for *non-file-like* entries
	readSize uint64
}

func newFile(p *dir, e plugin.Entry) *file {
	return &file{fuseNode: newFuseNode("f", p, e), writers: make(map[fuse.HandleID]struct{})}
}

func (f *file) isFileLikeEntry() bool {
	attr := plugin.Attributes(f.entry)
	return attr.HasSize()
}

// If currently writing a file-like object, we should use local content to fulfil many requests.
// Returning false means the entry is not file-like OR we're not writing to it.
func (f *file) useLocalContent() bool {
	return len(f.writers) != 0 && f.isFileLikeEntry()
}

var _ = fs.Node(&file{})
var _ = fs.Handle(&file{})

func (f *file) Attr(ctx context.Context, a *fuse.Attr) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	if !f.useLocalContent() {
		// Fetch updated attributes only if we're not currently writing to it.
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
	applyAttr(a, attr, defaultMode(f.entry))

	if f.useLocalContent() || !f.isFileLikeEntry() {
		// Use whatever size we know locally. Retrieving content can be expensive so we settle for
		// including size only when it's been retreived previously by opening the file.
		a.Size = f.readSize
	}
}

func defaultMode(entry plugin.Entry) os.FileMode {
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
// opened WriteOnly. That only happens when performing a partial write of a *file-like* entry.
func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	f.mux.Lock()
	defer f.mux.Unlock()
	activity.Record(ctx, "FUSE: Open %v: %+v", f, *req)

	if !f.useLocalContent() {
		// Check for an updated entry in case it has static content, like for preloaded external plugin entries.
		entry, err := f.refind(ctx)
		if err != nil {
			activity.Warnf(ctx, "FUSE: Open errored %v, %v", f, err)
			return nil, err
		}
		f.entry = entry
	}

	readable := plugin.ReadAction().IsSupportedOn(f.entry)
	writable := plugin.WriteAction().IsSupportedOn(f.entry)
	switch {
	case req.Flags.IsReadOnly() && !readable:
		activity.Warnf(ctx, "FUSE: Open read-only unsupported on %v", f)
		return nil, syscall.ENOTSUP
	case req.Flags.IsWriteOnly() && !writable:
		activity.Warnf(ctx, "FUSE: Open write-only unsupported on %v", f)
		return nil, syscall.ENOTSUP
	case req.Flags.IsReadWrite() && (!readable || !writable):
		activity.Warnf(ctx, "FUSE: Open read-write unsupported on %v", f)
		return nil, syscall.ENOTSUP
	}

	if !f.isFileLikeEntry() && req.Flags.IsReadWrite() {
		// Error ReadWrite on non-file-like entries because it probably won't work well.
		activity.Warnf(ctx, "FUSE: Open Read/Write is not supported on non-file-like entry %v", f)
		return nil, syscall.ENOTSUP
	}

	if !f.isFileLikeEntry() {
		// Open the file in direct IO mode to avoid the kernel page cache. This also enables FUSE to
		// still read the entry's content so that built-in tools like cat and grep still work.
		resp.Flags |= fuse.OpenDirectIO
	}

	if f.isFileLikeEntry() || req.Flags.IsReadOnly() {
		// Get the entry's readable size if we expect to do any reads or keep a local representation.
		size, err := plugin.Size(ctx, f.entry)
		if err != nil {
			return nil, err
		}
		f.readSize = size
	}

	return f, nil
}

func (f *file) releaseWriter(handle fuse.HandleID) {
	if _, ok := f.writers[handle]; ok {
		delete(f.writers, handle)

		if len(f.writers) == 0 {
			// If we just released the last writer, release the data buffer to conserve memory and
			// invalidate cache on the entry and its parent so we get updated content and size on the
			// next request. Leave size for entries that don't set it.
			f.data = nil
			plugin.ClearCacheFor(plugin.ID(f.entry), true)
		}
	}
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

	// Release writer and cleanup if all writers are released. Note that this is usually a noop for
	// non-file-like entries, they will have released the writers immediately after `plugin.Write`.
	f.releaseWriter(req.Handle)

	activity.Record(ctx, "FUSE: Release %v: %+v", f, *req)
	return nil
}

var _ = fs.HandleReader(&file{})

func (f *file) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.useLocalContent() {
		fuseutil.HandleRead(req, resp, f.data)
	} else {
		data, err := plugin.ReadWithAnalytics(ctx, f.entry, int64(req.Size), req.Offset)
		if err != nil && err != io.EOF {
			// If we don't ignore EOF, then cat will display an input/output error message
			// for entries with unknown content size.
			return err
		}
		resp.Data = data
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

	if f.isFileLikeEntry() {
		// If starting write beyond the current length, read to fill it in.
		if start := int64(len(f.data)); req.Offset > start {
			data, err := f.load(ctx, start, req.Offset)
			if err != nil {
				return err
			}
			f.data = append(f.data, data...)
		}
	}

	// Expand the buffer if necessary to store the write data.
	newLen := req.Offset + int64(len(req.Data))
	if newLen := int(newLen); newLen > len(f.data) {
		f.data = append(f.data, make([]byte, newLen-len(f.data))...)
	}

	// If file-like, then update readable size to reflect the expanded buffer.
	if f.isFileLikeEntry() && f.readSize < uint64(newLen) {
		f.readSize = uint64(newLen)
	}

	resp.Size = copy(f.data[req.Offset:], req.Data)
	activity.Record(ctx, "FUSE: Write %v/%v bytes starting at %v from %v", resp.Size, len(req.Data), req.Offset, f)
	return nil
}

func (f *file) load(ctx context.Context, start, end int64) ([]byte, error) {
	if !f.isFileLikeEntry() {
		panic("load called on non-file-like entry")
	}

	if !plugin.ReadAction().IsSupportedOn(f.entry) {
		activity.Warnf(ctx, "FUSE: Non-contiguous writes (at %v) unsupported on %v", start, f)
		return nil, syscall.ENOTSUP
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

	if _, ok := f.writers[req.Handle]; !ok {
		return nil
	}

	// If this handle had an open writer, write current data.
	dataLen := int64(len(f.data))
	if f.isFileLikeEntry() {
		// Only file-like entries keep data and readSize in sync.
		if uint64(dataLen) > f.readSize {
			panic("Size was not kept up-to-date with changes to data.")
		}

		if uint64(dataLen) < f.readSize {
			// Missing some data, load the remainder before writing.
			data, err := f.load(ctx, dataLen, int64(f.readSize))
			if err != nil && err != io.EOF {
				return err
			}
			f.data = append(f.data, data...)

			// If a call to `Setattr` was used to increase the file's size, then `load` will have
			// returned EOF and the loaded data would not be enough to increase the local content buffer
			// to `readSize`. Fill the rest with null characters.
			if sz := uint64(len(f.data)); sz < f.readSize {
				f.data = append(f.data, make([]byte, f.readSize-sz)...)
			}
		}
	}

	if err := plugin.WriteWithAnalytics(ctx, f.entry.(plugin.Writable), f.data); err != nil {
		activity.Warnf(ctx, "FUSE: Error writing %v: %v", f, err)
		return err
	}

	// Non-file-like entries start from scratch on each Write operation, and have their cache
	// invalidated whenever we write to them because we can't accurately model their readable data.
	if !f.isFileLikeEntry() {
		f.releaseWriter(req.Handle)
	}
	return nil
}

var _ = fs.NodeSetattrer(&file{})

func (f *file) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	f.mux.Lock()
	defer f.mux.Unlock()
	activity.Record(ctx, "FUSE: Setattr[%v] %v: %+v", req.Handle, f, *req)

	if req.Valid.Size() {
		if !req.Valid.Handle() {
			// No guarantee we'll ever write the change. If this is ever necessary, we could update it
			// to immediately do a plugin.Write.
			return syscall.ENOTSUP
		}

		// Ensure handle is in list of writers because we need to operate on a local copy of the data,
		// and changing the file size is similar to initiating a write.
		f.writers[req.Handle] = struct{}{}

		if f.isFileLikeEntry() {
			// Update known size.
			f.readSize = req.Size
		} else {
			// Non-file-like entries use `data` as a write buffer. There's nothing to fill in from, so
			// just resize as necessary.
			if curLen := uint64(len(f.data)); req.Size > curLen {
				f.data = append(f.data, make([]byte, req.Size-uint64(len(f.data)))...)
			} else if req.Size < curLen {
				f.data = f.data[:req.Size]
			}
		}
	}

	f.fillAttr(&resp.Attr)
	return nil
}

// Needs to be defined or vim gets an EIO error on Fsync.
var _ = fs.NodeFsyncer(&file{})

func (f *file) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	// As noted in the docs for fs.NodeFsyncer, this should be implemented on a Handle. Write Fsync
	// should be unnecessary because Flush handles complete serialization out. On a handle opened
	// for reading, we could potentially invalidate the Wash cache and re-request data from the
	// plugin, but in most cases that doesn't seem to be necessary.
	activity.Record(ctx, "FUSE: Fsync %v: %+v", f, *req)
	return syscall.ENOSYS
}

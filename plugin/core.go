package plugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/log"
)

var slow = false

// DefaultTimeout is a default cache timeout.
const DefaultTimeout = 10 * time.Second

// Init sets up plugin core configuration on startup.
func Init(_slow bool) {
	slow = _slow
}

// ==== Plugin registry (FS) ====
//
// Here we implement the DirProtocol interface for FS so
// that FUSE can recognize it as a valid root directory
//

// Root presents the root of the filesystem.
func (f *FS) Root() (fs.Node, error) {
	log.Printf("Entering root of filesystem")
	return &Dir{f}, nil
}

// Find the named item or return nil.
func (f *FS) Find(_ context.Context, name string) (Node, error) {
	if client, ok := f.Plugins[name]; ok {
		return &Dir{client}, nil
	}
	return nil, ENOENT
}

// List all clients as directories.
func (f *FS) List(_ context.Context) ([]Node, error) {
	keys := make([]Node, 0, len(f.Plugins))
	for _, v := range f.Plugins {
		keys = append(keys, &Dir{v})
	}
	return keys, nil
}

// Name returns '/'.
func (f *FS) Name() string {
	return "/"
}

// Attr returns basic (zero) attributes for the root directory.
func (f *FS) Attr(ctx context.Context) (*Attributes, error) {
	// Only ever called with "/". Return latest Mtime of all clients.
	var latest time.Time
	for _, v := range f.Plugins {
		attr, err := v.Attr(ctx)
		if err != nil {
			return nil, err
		}
		if attr.Mtime.After(latest) {
			latest = attr.Mtime
		}
	}
	return &Attributes{Mtime: latest, Valid: 100 * time.Millisecond}, nil
}

// Xattr returns an empty map.
func (f *FS) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

// ==== FUSE Directory Interface ====

// NewDir creates a new Dir object.
func NewDir(impl DirProtocol) *Dir {
	return &Dir{impl}
}

func (d *Dir) String() string {
	if v, ok := d.DirProtocol.(fmt.Stringer); ok {
		return v.String()
	}
	return d.Name()
}

var startTime = time.Now()

// Applies attributes where non-default, and sets defaults otherwise.
func applyAttr(a *fuse.Attr, attr *Attributes) {
	a.Valid = 1 * time.Minute
	if attr.Valid != 0 {
		a.Valid = attr.Valid
	}

	// TODO: tie this to actual hard links in plugins
	a.Nlink = 1
	a.Mode = attr.Mode
	a.Size = attr.Size

	var zero time.Time
	a.Mtime = startTime
	if attr.Mtime != zero {
		a.Mtime = attr.Mtime
	}
	a.Atime = startTime
	if attr.Atime != zero {
		a.Atime = attr.Atime
	}
	a.Ctime = startTime
	if attr.Ctime != zero {
		a.Ctime = attr.Ctime
	}
	a.Crtime = startTime
}

// Getattr implements the NodeGetattrer interface.
func (d *Dir) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	log.Printf("Getattr[pid=%v] %v", req.Pid, d)
	return d.Attr(ctx, &resp.Attr)
}

// Attr returns the attributes of a directory.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	attr, err := d.DirProtocol.Attr(ctx)
	if err != nil {
		log.Printf("Error[Attr,%v]: %v", d, err)
	}
	if attr.Mode == 0 {
		attr.Mode = os.ModeDir | 0550
	}
	applyAttr(a, attr)
	log.Printf("Attr of dir %v: %v", d, a)
	return err
}

// Listxattr lists extended attributes for the resource.
func (d *Dir) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	xattrs, err := d.Xattr(ctx)
	if err != nil {
		log.Printf("Error[Listxattr,%v]: %v", d, err)
		return err
	}

	for k := range xattrs {
		resp.Append(k)
	}
	log.Printf("Listxattr[pid=%v] %v", req.Pid, d)
	return nil
}

// Getxattr gets extended attributes for the resource.
func (d *Dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	if req.Name == "com.apple.FinderInfo" {
		return nil
	}

	xattrs, err := d.Xattr(ctx)
	if err != nil {
		log.Printf("Error[Getxattr,%v,%v]: %v", d, req.Name, err)
		return err
	}

	resp.Xattr = xattrs[req.Name]
	log.Printf("Getxattr[pid=%v] %v", req.Pid, d)
	return nil
}

func prefetch(entry fs.Node) {
	if slow {
		return
	}

	switch v := entry.(type) {
	case *Dir:
		go func() { v.List(context.Background()) }()
	case *File:
		// TODO: This can be pretty expensive. Probably better to move it to individual implementations
		// where they can choose to do this if Attr is requested.
		go func() {
			buf, err := v.FileProtocol.Open(context.Background())
			if closer, ok := buf.(io.Closer); err == nil && ok {
				go func() {
					time.Sleep(DefaultTimeout)
					closer.Close()
				}()
			}
		}()
	default:
		log.Debugf("Not sure how to prefetch for %v", v)
	}
}

// Lookup searches a directory for children.
func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	entry, err := d.Find(ctx, req.Name)
	if err == nil {
		log.Printf("Find[pid=%v] %v", req.Pid, entry)
		prefetch(entry)
	} else {
		log.Printf("%v not found in %v", req.Name, d)
	}
	return entry, err
}

// ReadDirAll lists all children of the directory.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := d.List(ctx)
	if err != nil {
		log.Printf("Error[List,%v]: %v", d, err)
		return nil, err
	}

	log.Printf("List %v in %v", len(entries), d)

	res := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		var de fuse.Dirent
		switch v := entry.(type) {
		case *Dir:
			de.Name = v.Name()
			de.Type = fuse.DT_Dir
		case *File:
			de.Name = v.Name()
		}
		res[i] = de
	}
	return res, nil
}

// ==== FUSE File Interface ====

// NewFile creates a new Dir object.
func NewFile(impl FileProtocol) *File {
	return &File{impl}
}

func (f *File) String() string {
	if v, ok := f.FileProtocol.(fmt.Stringer); ok {
		return v.String()
	}
	return f.Name()
}

// Getattr implements the NodeGetattrer interface.
func (f *File) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	log.Printf("Getattr[pid=%v] %v", req.Pid, f)
	return f.Attr(ctx, &resp.Attr)
}

// Attr returns the attributes of a file.
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	attr, err := f.FileProtocol.Attr(ctx)
	if err != nil {
		log.Printf("Error[Attr,%v]: %v", f, err)
	}
	if attr.Mode == 0 {
		attr.Mode = 0440
	}
	applyAttr(a, attr)
	log.Printf("Attr of file %v: %v", f, a)
	return err
}

// Listxattr lists extended attributes for the resource.
func (f *File) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	xattrs, err := f.Xattr(ctx)
	if err != nil {
		log.Printf("Error[Listxattr,%v]: %v", f, err)
		return err
	}

	for k := range xattrs {
		resp.Append(k)
	}
	log.Printf("Listxattr[pid=%v] %v", req.Pid, f)
	return nil
}

// Getxattr gets extended attributes for the resource.
func (f *File) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	if req.Name == "com.apple.FinderInfo" {
		return nil
	}

	xattrs, err := f.Xattr(ctx)
	if err != nil {
		log.Printf("Error[Getxattr,%v,%v]: %v", f, req.Name, err)
		return err
	}

	resp.Xattr = xattrs[req.Name]
	log.Printf("Getxattr[pid=%v] %v", req.Pid, f)
	return nil
}

// Open a file for reading.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// Initiate content request and return a channel providing the results.
	log.Printf("Opening[pid=%v] %v", req.Pid, f)
	r, err := f.FileProtocol.Open(ctx)
	if err != nil {
		log.Printf("Error[Open,%v]: %v", f, err)
		return nil, err
	}
	log.Printf("Opened[pid=%v] %v", req.Pid, f)
	return &FileHandle{r: r, id: f.String()}, nil
}

// Release closes the open file.
func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Printf("Release[pid=%v] %v", req.Pid, fh.id)
	if closer, ok := fh.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Read fills a buffer with the requested amount of data from the file.
func (fh *FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	buf := make([]byte, req.Size)
	n, err := fh.r.ReadAt(buf, req.Offset)
	if err == io.EOF {
		err = nil
	}
	log.Printf("Read[pid=%v] %v, %v/%v bytes starting at %v: %v", fh.id, req.Pid, n, req.Size, req.Offset, err)
	resp.Data = buf[:n]
	return err
}

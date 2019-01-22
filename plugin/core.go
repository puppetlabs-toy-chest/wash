package plugin

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// Root presents the root of the filesystem.
func (f *FS) Root() (fs.Node, error) {
	log.Println("Entering root of filesystem")
	return &Dir{
		Client: f,
		Name:   "/",
	}, nil
}

// Find the named item or return nil.
func (f *FS) Find(_ context.Context, parent *Dir, name string) (Node, error) {
	if client, ok := f.Clients[name]; ok {
		log.Printf("Found client %v: %v", name, client)
		return &Dir{client, parent, name}, nil
	}
	log.Printf("Client %v not found", name)
	return nil, ENOENT
}

// List all clients as directories.
func (f *FS) List(_ context.Context, parent *Dir) ([]Node, error) {
	log.Printf("Listed %v clients in /", len(f.Clients))
	keys := make([]Node, 0, len(f.Clients))
	for k, v := range f.Clients {
		keys = append(keys, &Dir{v, parent, k})
	}
	return keys, nil
}

// Attr returns basic (zero) attributes for the root directory.
func (f *FS) Attr(ctx context.Context, name string) (*Attributes, error) {
	// Only ever called with "/". Return latest Mtime of all clients.
	var latest time.Time
	for k, v := range f.Clients {
		attr, err := v.Attr(ctx, k)
		if err != nil {
			return nil, err
		}
		if attr.Mtime.After(latest) {
			latest = attr.Mtime
		}
	}
	return &Attributes{Mtime: latest, Valid: 10 * time.Second}, nil
}

// Xattr returns an empty map.
func (f *FS) Xattr(ctx context.Context, name string) (map[string][]byte, error) {
	data := make(map[string][]byte)
	return data, nil
}

// Attr returns the attributes of a directory.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0550

	attr, err := d.Client.Attr(ctx, d.Name)
	a.Mtime = attr.Mtime
	a.Size = attr.Size
	a.Valid = attr.Valid
	log.Printf("Attr of dir %v: %v, %v", d.Name, a.Mtime, a.Size)
	return err
}

// Lookup searches a directory for children.
func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	return d.Client.Find(ctx, d, req.Name)
}

// ReadDirAll lists all children of the directory.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := d.Client.List(ctx, d)
	if err != nil {
		return nil, err
	}

	res := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		var de fuse.Dirent
		switch v := entry.(type) {
		case *Dir:
			de.Name = v.Name
			de.Type = fuse.DT_Dir
		case *File:
			de.Name = v.Name
		}
		res[i] = de
	}
	return res, nil
}

// Attr returns the attributes of a file.
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0440

	attr, err := f.Client.Attr(ctx, f.Name)
	a.Mtime = attr.Mtime
	a.Size = attr.Size
	a.Valid = attr.Valid
	log.Printf("Attr of file %v: %v, %s", f.Name, a.Size, a.Mtime)
	return err
}

// Listxattr lists extended attributes for the resource.
func (f *File) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	xattrs, err := f.Client.Xattr(ctx, f.Name)
	if err != nil {
		return err
	}

	for k := range xattrs {
		resp.Append(k)
	}
	return nil
}

// Getxattr gets extended attributes for the resource.
func (f *File) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	xattrs, err := f.Client.Xattr(ctx, f.Name)
	if err != nil {
		return err
	}

	resp.Xattr = xattrs[req.Name]
	return nil
}

// Open a file for reading.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// Initiate content request and return a channel providing the results.
	r, err := f.Client.Open(ctx, f.Name)
	if err != nil {
		return nil, err
	}
	log.Printf("Opened %v", f.Name)
	return &FileHandle{r: r}, nil
}

// Release closes the open file.
func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	switch v := fh.r.(type) {
	case io.Closer:
		return v.Close()
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
	log.Printf("Read %v/%v bytes starting at %v: %v", n, req.Size, req.Offset, err)
	resp.Data = buf[:n]
	return err
}

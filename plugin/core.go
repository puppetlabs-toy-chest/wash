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
		client: f,
		name:   "/",
	}, nil
}

// Find the named item or return nil.
func (f *FS) Find(_ context.Context, name string) (*Entry, error) {
	if client, ok := f.Clients[name]; ok {
		log.Printf("Found client %v: %v", name, client)
		return &Entry{
			Client: client,
			Name:   name,
			Isdir:  true,
		}, nil
	}
	log.Printf("Client %v not found", name)
	return nil, ENOENT
}

// List all clients as directories.
func (f *FS) List(context.Context) ([]Entry, error) {
	log.Printf("Listed %v clients in /", len(f.Clients))
	keys := make([]Entry, 0, len(f.Clients))
	for k, v := range f.Clients {
		keys = append(keys, Entry{Client: v, Name: k, Isdir: true})
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
	return &Attributes{Mtime: latest}, nil
}

const validFor = 100 * time.Millisecond

// Attr returns the attributes of a directory.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0550
	a.Valid = validFor

	attr, err := d.client.Attr(ctx, d.name)
	a.Mtime = attr.Mtime
	a.Size = attr.Size
	log.Printf("Attr of dir %v: %v, %v", d.name, a.Mtime, a.Size)
	return err
}

// Lookup searches a directory for children.
func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	entry, err := d.client.Find(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if entry.Isdir {
		return &Dir{client: entry.Client.(DirProtocol), name: entry.Name}, nil
	}
	return &File{client: entry.Client.(FileProtocol), name: entry.Name}, nil
}

// ReadDirAll lists all children of the directory.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := d.client.List(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]fuse.Dirent, len(entries))
	for i, entry := range entries {
		var de fuse.Dirent
		de.Name = entry.Name
		if entry.Isdir {
			de.Type = fuse.DT_Dir
		}
		res[i] = de
	}
	return res, nil
}

// Attr returns the attributes of a file.
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0440
	a.Valid = validFor

	attr, err := f.client.Attr(ctx, f.name)
	a.Mtime = attr.Mtime
	a.Size = attr.Size
	log.Printf("Attr of file %v: %v, %s", f.name, a.Size, a.Mtime)
	return err
}

// Open a file for reading.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// Initiate content request and return a channel providing the results.
	r, err := f.client.Open(ctx, f.name)
	if err != nil {
		return nil, err
	}
	log.Printf("Opened %v", f.name)
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

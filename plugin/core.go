package plugin

import (
	"context"
	"io"
	"io/ioutil"
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
	log.Printf("Listing %v clients in /", len(f.Clients))
	keys := make([]Entry, 0, len(f.Clients))
	for k, v := range f.Clients {
		keys = append(keys, Entry{Client: v, Name: k, Isdir: true})
	}
	return keys, nil
}

// Attr returns the attributes of a directory.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Printf("Checking attr of dir %v", d.name)
	a.Mode = os.ModeDir | 0550

	// This is a hack to suggest content may update every second.
	// TODO: need to base it off an actual request to check whether there's new data.
	a.Mtime = time.Now()
	a.Valid = 1 * time.Second
	return nil
}

// Lookup searches a directory for children.
func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	entry, err := d.client.Find(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if entry.Isdir {
		return &Dir{client: entry.Client.(GroupTraversal), name: entry.Name}, nil
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
	log.Printf("Checking attr of file %v", f.name)
	a.Mode = 0440

	// Read the content to figure out how large it is.
	r, err := f.client.Read(ctx, f.name)
	if err != nil {
		return err
	}
	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	a.Size = uint64(len(buf))

	// This is a hack to suggest the logs may update every second.
	// TODO: need to base it off an actual request to check whether there's new data.
	a.Mtime = time.Now()
	a.Valid = 1 * time.Second
	return nil
}

// Open a file for reading. Not yet supported.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// Initiate content request and return a channel providing the results.
	log.Println("Reading content of", f.name)
	r, err := f.client.Read(ctx, f.name)
	if err != nil {
		return nil, err
	}
	r.Close()
	return &FileHandle{client: f.client, name: f.name}, nil
}

// Release closes the open file.
func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return nil
}

// Read fills a buffer with the requested amount of data from the file.
func (fh *FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	log.Printf("Reading %v bytes starting at %v", req.Size, req.Offset)
	r, err := fh.client.Read(ctx, fh.name)
	if err != nil {
		return err
	}
	defer r.Close()

	// Skip the offset
	buf := make([]byte, req.Offset)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}

	buf = make([]byte, req.Size)
	n, err := io.ReadFull(r, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	log.Printf("Read %v bytes: %v", n, err)
	resp.Data = buf[:n]
	return err
}

package plugin

import (
	"context"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// Root presents the root of the filesystem.
func (f *FS) Root() (fs.Node, error) {
	n := &Dir{
		client: f,
		name:   "/",
	}
	return n, nil
}

// Find the named item or return nil.
func (f *FS) Find(name string) (Node, error) {
	if client, ok := f.Clients[name]; ok {
		return &Dir{
			client: client,
			name:   name,
		}, nil
	}
	return nil, ENOENT
}

// List all clients as directories.
func (f *FS) List() ([]Entry, error) {
	keys := make([]Entry, 0, len(f.Clients))
	for k := range f.Clients {
		keys = append(keys, Entry{Name: k, Isdir: true})
	}
	return keys, nil
}

// Attr returns the attributes of a directory.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0550
	return nil
}

// Lookup searches a directory for children.
func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	return d.client.Find(req.Name)
}

// ReadDirAll lists all children of the directory.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := d.client.List()
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
	return nil
}

// Open a file for reading. Not yet supported.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return nil, fuse.ENOTSUP
}

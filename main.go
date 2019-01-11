package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/docker"
)

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(progName + ": ")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	if err := mount(mountpoint); err != nil {
		log.Fatal(err)
	}
}

func mount(mountpoint string) error {
	dockercli, err := docker.Create()
	if err != nil {
		return err
	}

	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	filesys := &FS{
		docker: dockercli,
	}
	if err := fs.Serve(c, filesys); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	return nil
}

// FS contains the core filesystem data.
type FS struct {
	docker *docker.Client
}

var _ fs.FS = (*FS)(nil)

// Root presents the root of the filesystem.
func (f *FS) Root() (fs.Node, error) {
	n := &Dir{
		client: f,
		name:   "/",
	}
	return n, nil
}

// Dir represents a directory within the system, with the client
// necessary to represent it and the full path to the directory.
type Dir struct {
	client interface{}
	name   string
}

var _ fs.Node = (*Dir)(nil)

// Attr returns the attributes of a directory.
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0550
	return nil
}

var _ = fs.NodeRequestLookuper(&Dir{})

// Lookup searches a directory for children.
func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	switch v := d.client.(type) {
	case *FS:
		if req.Name == "docker" {
			return &Dir{
				client: v.docker,
				name:   d.name + req.Name,
			}, nil
		}
	case *docker.Client:
		containers, err := v.List()
		if err != nil {
			return nil, err
		}
		for _, container := range containers {
			if container.ID == req.Name {
				return &File{meta: container}, nil
			}
		}
	}
	return nil, fuse.ENOENT
}

var _ = fs.HandleReadDirAller(&Dir{})

// ReadDirAll lists all children of the directory.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var res []fuse.Dirent
	switch v := d.client.(type) {
	case *FS:
		var de fuse.Dirent
		de.Name = "docker"
		de.Type = fuse.DT_Dir
		res = append(res, de)
	case *docker.Client:
		containers, err := v.List()
		if err != nil {
			return nil, err
		}

		for _, container := range containers {
			var de fuse.Dirent
			de.Name = container.ID
			res = append(res, de)
		}
	}
	return res, nil
}

// File contains metadata about the file.
type File struct {
	meta interface{}
}

var _ fs.Node = (*File)(nil)

// Attr returns the attributes of a file.
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0440
	return nil
}

var _ = fs.NodeOpener(&File{})

// Open a file for reading. Not yet supported.
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return nil, fuse.ENOTSUP
}

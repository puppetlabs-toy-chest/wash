package plugin

import (
	"context"
	"io"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// Node represents a filesystem node
type Node = fs.Node

// ENOENT states the entity does not exist
const ENOENT = fuse.ENOENT

// Entry in a filesystem
type Entry struct {
	Client interface{}
	Name   string
	Isdir  bool
}

// GroupTraversal that plugins are expected to model.
type GroupTraversal interface {
	Find(ctx context.Context, name string) (*Entry, error)
	List(ctx context.Context) ([]Entry, error)
}

// Content protocol.
type Content interface {
	Read(ctx context.Context, name string) (io.ReadCloser, error)
}

// FS contains the core filesystem data.
type FS struct {
	Clients map[string]GroupTraversal
}

var _ fs.FS = (*FS)(nil)

// Dir represents a directory within the system, with the client
// necessary to represent it and the full path to the directory.
type Dir struct {
	client GroupTraversal
	name   string
}

var _ fs.Node = (*Dir)(nil)
var _ = fs.NodeRequestLookuper(&Dir{})
var _ = fs.HandleReadDirAller(&Dir{})

// FileProtocol is protocols expected of a File resource.
type FileProtocol interface {
	GroupTraversal
	Content
}

// File contains metadata about the file.
type File struct {
	client FileProtocol
	name   string
}

var _ fs.Node = (*File)(nil)
var _ = fs.NodeOpener(&File{})

// FileHandle contains an IO object that can be read.
type FileHandle struct {
	r io.ReadCloser
}

var _ fs.Handle = (*FileHandle)(nil)
var _ = fs.HandleReleaser(&FileHandle{})
var _ = fs.HandleReader(&FileHandle{})

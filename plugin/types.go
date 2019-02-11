package plugin

import (
	"context"
	"io"
	"os"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// ==== Wash Protocols and Resources ====

// Entry is a basic named resource type
type Entry interface{ Name() string }

// EntryT implements Entry, making it easy to create new named types.
type EntryT struct{ EntryName string }

// Name returns the entry's name.
func (e EntryT) Name() string { return e.EntryName }

// Resource is an entry that has metadata.
type Resource interface {
	Entry
	Metadata(context.Context) (interface{}, error)
}

// Group is an entry that can list its contents, an array of entries.
// It will be represented as a directory in the wash filesystem.
type Group interface {
	Entry
	LS(context.Context) ([]Entry, error)
}

// Execable is an entry that can have a command run on it.
type Execable interface {
	Exec(context.Context, string) (io.Reader, error)
}

// File is an entry that specifies filesystem attributes.
type File interface {
	Entry
	Attr() Attributes
}

// Pipe is an entry that returns a stream of updates. It will be represented
// as a named pipe (FIFO) in the wash filesystem.
type Pipe interface {
	Stream(context.Context) (io.Reader, error)
}

// Readable is an entry that has a fixed amount of content we can read.
type Readable interface {
	Size() uint64
	Open(context.Context) (io.ReaderAt, error)
}

// Writable is an entry that we can write new data to.
type Writable interface {
	Save(context.Context, io.Reader) error
}

// Attributes of resources.
type Attributes struct {
	Atime time.Time
	Mtime time.Time
	Ctime time.Time
	Mode  os.FileMode
	Size  uint64
	Valid time.Duration
}

// ==== FUSE Adapters ====

// Node represents a filesystem node
type Node = fs.Node

// ENOENT states the entity does not exist
const (
	ENOENT  = fuse.ENOENT
	ENOTSUP = fuse.ENOTSUP
)

// The Registry contains the core filesystem data.
// Plugins: maps plugin mount points to their implementations.
type Registry struct {
	Plugins map[string]Entry
}

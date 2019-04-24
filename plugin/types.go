/*
Package plugin defines a set of interfaces that plugins must implement to enable wash
functonality.

All resources must implement the Entry interface. To do so they should include the EntryBase
type, and initialize it via NewEntry. For example
	type myResource struct {
		plugin.EntryBase
	}
	...
	rsc := myResource{plugin.NewEntry("a resource")}
EntryBase gives the resource a name - which is how it will be displayed in the filesystem
or referenced via the API - and tools for controlling how its data is cached.

The Group interface identifies the resource as a container for other things. Implementing it
enables displaying it as a directory in the filesystem. Anything that does not implement
Group will be displayed as a file.

The Readable interface gives a file its contents when read via the filesystem.

All of the above, as well as other types - Resource, Execable, Pipe - provide
additional functionality via the HTTP API.
*/
package plugin

// This file should be reserved for types that plugin authors need to understand.

import (
	"context"
	"io"
	"time"
)

// Entry is the interface for things that are representable by Wash's filesystem. This includes
// plugin roots; resources like containers and volumes; placeholders (e.g. the containers directory
// in the Docker plugin); read-only files like the metadata.json files for containers and EC2
// instances; and more. It is a sealed interface, meaning you must use plugin.NewEntry when
// creating your plugin objects.
//
// Metadata returns a complete description of the entry. See the EntryBase documentation for more
// details on when to override it.
type Entry interface {
	Metadata(ctx context.Context) (EntryMetadata, error)
	name() string
	attributes() EntryAttributes
	slashReplacementChar() rune
	id() string
	setID(id string)
	getTTLOf(op defaultOpCode) time.Duration
}

// Group is an entry that can list its contents, an array of entries.
// It will be represented as a directory in the wash filesystem.
type Group interface {
	Entry
	List(context.Context) ([]Entry, error)
}

// Root is the root object of a plugin.
type Root interface {
	Group
	Init() error
}

// ExecOptions is a struct we can add new features to that must be serializable to JSON.
// Examples of potential features: user, privileged, tty, map of environment variables, string of stdin, timeout.
type ExecOptions struct {
	Stdin io.Reader
}

// ExecOutputChunk is a struct containing a chunk of the Exec'ed cmd's output.
// For StreamID, 0 = stdout and 1 = stderr.
type ExecOutputChunk struct {
	StreamID  int8
	Timestamp time.Time
	Data      string
	Err       error
}

// ExecResult is a struct that contains the result of invoking Execable#exec.
// Any of these fields can be nil. The OutputCh will be closed when execution completes.
type ExecResult struct {
	OutputCh   <-chan ExecOutputChunk
	ExitCodeCB func() (int, error)
}

// Execable is an entry that can have a command run on it.
type Execable interface {
	Entry
	Exec(ctx context.Context, cmd string, args []string, opts ExecOptions) (ExecResult, error)
}

// Streamable is an entry that returns a stream of updates.
type Streamable interface {
	Entry
	Stream(context.Context) (io.Reader, error)
}

// SizedReader returns a ReaderAt that can report its Size.
type SizedReader interface {
	io.ReaderAt
	Size() int64
}

// Readable is an entry that has a fixed amount of content we can read.
type Readable interface {
	Entry
	Open(context.Context) (SizedReader, error)
}

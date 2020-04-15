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

Implementing the Parent interface displays that resource as a directory on the filesystem.
Anything that does not implement Parent will be displayed as a file.

The Readable interface allows reading data from an entry via the filesystem. The Writable
interface allows sending data to the entry.

Wash distinguishes between two different patterns for things you can read and write. It considers
a "file-like" entry to be one with a defined size (so the `size` attribute is set when listing the
entry). Reading and writing a "file-like" entry edits the contents. Something that can be read and
written but doesn't define size has different characteristics. Reading and writing are not
symmetrical: if you write to it then read from it, you may not see what you just wrote. So these
non-file-like entries error if you try to open them with a ReadWrite handle. If your plugin
implements non-file-like write-semantics, remember to document how they work in the plugin schema's
description.

All of the above, as well as other types - Execable, Stream - provide additional functionality
via the HTTP API.
*/
package plugin

// This file should be reserved for types that plugin authors need to understand.

import (
	"context"
	"io"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

// Entry is the interface for things that are representable by Wash's filesystem. This includes
// plugin roots; resources like containers and volumes; placeholders (e.g. the containers directory
// in the Docker plugin); read-only files like the metadata.json files for containers and EC2
// instances; and more. It is a sealed interface, meaning you must use plugin.NewEntry when
// creating your plugin objects.
//
// Metadata returns a complete description of the entry. See the EntryBase documentation for more
// details on when to override it.
//
// Schema returns the entry's schema. See plugin.EntrySchema for more details.
type Entry interface {
	Metadata(ctx context.Context) (JSONObject, error)
	Schema() *EntrySchema
	// eb => entryBase
	eb() *EntryBase
}

/*
Parent is an entry with children. It will be represented as a directory in the Wash filesystem.
All parents must implement ChildSchemas, which is useful for documenting your plugin's hierarchy
via the stree command, and for optimizing `wash find`'s traversal by eliminating non-satisfying
paths. Your implementation of ChildSchemas should be something like
	return []*plugin.EntrySchema{
		(&childObj1{}).Schema(),
		(&childObj2{}).Schema(),
		...
		(&childObjN{}).Schema(),
	}

For example,
	return []*plugin.EntrySchema{
		(&containerLogFile{}).Schema(),
		(&containerMetadata{}).Schema(),
		(&vol.FS{}).Schema(),
	}
is the implementation of ChildSchemas for a Docker container.
*/
type Parent interface {
	Entry
	// TODO: "nil" schemas mean that the schema's unknown. This condition's necessary for
	// the plugin registry b/c external plugins may not have a schema. However, core plugins
	// should never return a "nil" schema -- their schema's always known. Since this specification
	// affects core plugin authors, it should be cleaned up once the metadata schema + external
	// plugin schema work is finished.
	ChildSchemas() []*EntrySchema
	List(context.Context) ([]Entry, error)
}

// SchemaMap represents a map of <type> => <JSON schema>.
type SchemaMap = map[interface{}]*JSONSchema

// Root represents the plugin root. The Init function is passed a config map representing
// plugin-specific configuration.
type Root interface {
	Parent
	Init(map[string]interface{}) error
}

// HasWrappedTypes is an interface that's used by the EntrySchema#SetMeta*Schema methods to return
// the right metadata schema for wrapped types. Plugin roots should implement this interface if the
// plugin's SDK wraps primitive types like date, integer, number, string, boolean, etc. See
// kubernetes/root.go for an example of when WrappedTypes is used.
//
// NOTE: Type aliases like "type Time = time.Time" do NOT count as wrapped types. Pointers
// also don't count. Only promoted types ("type Time time.Time") and wrapper structs
// ("struct Time { t time.Time }") count.
//
// NOTE: Without WrappedTypes, the underlying JSON schema library will treat promoted types and
// wrapper structs as JSON objects, which would be incorrect. This, unfortunately, is a
// limitation of Go's reflect package.
type HasWrappedTypes interface {
	Root
	WrappedTypes() SchemaMap
}

// ExecOptions is a struct we can add new features to that must be serializable to JSON.
// Examples of potential features: user, privileged, map of environment variables, timeout.
type ExecOptions struct {
	// Stdin can be used to pass a stream of input to write to stdin when executing the command.
	// It is not included in ExecOption's JSON serialization.
	Stdin io.Reader `json:"-"`

	// Tty instructs the executor to allocate a TTY (pseudo-terminal), which lets Wash communicate
	// with the running process via its Stdin. The TTY is used to send a process termination signal
	// (Ctrl+C) via Stdin when the passed-in Exec context is cancelled.
	//
	// NOTE TO PLUGIN AUTHORS: The Tty option is only relevant for executors that do not have an API
	// endpoint to stop a running command (e.g. Docker, Kubernetes). If your executor does have an
	// API endpoint to stop a running command, then ignore the Tty option. Note that the reason we
	// make Tty an option instead of having the relevant executors always attach a TTY is because
	// attaching a TTY can change the behavior of the command that's being executed.
	//
	// NOTE TO CALLERS: The Tty option is useful for executing your own stream-like commands (e.g.
	// tail -f), because it ensures that there are no orphaned processes after the request is
	// cancelled/finished.
	Tty bool `json:"tty"`

	// Elevate execution to run as a privileged user if not already running as a privileged user.
	Elevate bool `json:"elevate"`

	// WorkingDir is the directory in which to execute the command.
	WorkingDir string
}

// ExecPacketType identifies the packet type.
type ExecPacketType = string

// Enumerates packet types.
const (
	Stdout ExecPacketType = "stdout"
	Stderr ExecPacketType = "stderr"
)

// ExecOutputChunk is a struct containing a chunk of the Exec'ed cmd's output.
type ExecOutputChunk struct {
	StreamID  ExecPacketType
	Timestamp time.Time
	Data      string
	Err       error
}

// ExecCommand represents a command that was invoked by a call to Exec.
// It is a sealed interface, meaning you must use plugin.NewExecCommand
// to create instances of these objects.
//
// OutputCh returns a channel containing timestamped chunks of the command's
// stdout/stderr.
//
// ExitCode returns the command's exit code. It will block until the command's
// exit code is set, or until the execution context is cancelled. ExitCode will
// return an error if it fails to fetch the command's exit code.
type ExecCommand interface {
	OutputCh() <-chan ExecOutputChunk
	ExitCode() (int, error)
}

// Execable is an entry that can have a command run on it.
type Execable interface {
	Entry
	Exec(ctx context.Context, cmd string, args []string, opts ExecOptions) (ExecCommand, error)
}

// Streamable is an entry that returns a stream of updates.
type Streamable interface {
	Entry
	Stream(context.Context) (io.ReadCloser, error)
}

// BlockReadable is an entry with data that can be read in blocks.
// A BlockReadable entry must set its Size attribute. If you don't set it, the
// file size will be reported as 0 and reads will return an empty file.
type BlockReadable interface {
	Entry
	Read(ctx context.Context, size int64, offset int64) ([]byte, error)
}

// Readable is an entry with data that can be read.
type Readable interface {
	Entry
	Read(context.Context) ([]byte, error)
}

// Writable is an entry that we can write new data to. What that means can be
// implementation-specific; it could be overwriting a file, submitting a
// configuration change to an API, or writing data to a queue. It doesn't
// support a concept of a partial write.
//
// Writable can be implemented with or without Readable/BlockReadable. If an
// entry is only Writable, then only full writes (starting from offset 0) are
// allowed, anything else initiated by the filesystem will result in an error.
type Writable interface {
	Entry
	Write(context.Context, []byte) error
}

// Deletable is an entry that can be deleted. Entries that implement Delete
// should ensure that it and all its children are removed. If the entry has
// any dependencies that need to be deleted, then Delete should return an
// error.
//
// If Delete returns true, then that means the entry was deleted. If Delete
// returns false, then that means the entry is marked for deletion by the
// plugin's API. You should return false if you anticipate delete taking a long
// time (> 30 seconds).
type Deletable interface {
	Entry
	Delete(context.Context) (bool, error)
}

// Signalable is an entry that can be signaled. Signal should return nil if the
// signal was successfully sent. Otherwise, it should return an error explaining
// why the signal was not sent.
//
// NOTE: You can assume that the sent signal is downcased and valid.
type Signalable interface {
	Entry
	Signal(context.Context, string) error
}

// This interface exists to break the circular dependency between plugin and external.
// The external plugin implementation is in its own module so it can use other modules
// that implement new features and have dependencies on this module.
type externalPlugin interface {
	MethodSignature(string) MethodSignature
	// Entry#Schema's type-signature only makes sense for core plugins
	// since core plugin schemas do not require any error-prone API
	// calls. External plugin schemas can be prefetched (no error)
	// or obtained by shelling out (error-prone). Since the latter
	// operation is error prone, the type-signature of external plugin
	// schemas will include an error object. Since this is something
	// specific to external plugins, it makes sense to include the
	// error-prone version of schema here, in the Entry interface.
	SchemaGraph() (*linkedhashmap.Map, error)
	RawTypeID() string
	// Go doesn't allow overloaded functions, so the external plugin entry type
	// cannot implement both BlockReadable#Read and Readable#Read. Thus, external
	// plugins implement the BlockReadable interface via a separate BlockRead
	// method.
	BlockRead(ctx context.Context, size int64, offset int64) ([]byte, error)
}

package volume

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// FS presents a view of the filesystem of an Execable resource.
type FS struct {
	plugin.EntryBase
	executor plugin.Execable
	maxdepth int
}

// NewFS creates a new FS entry with the given name, using the supplied executor to satisfy volume
// operations.
func NewFS(name string, executor plugin.Execable, maxdepth int) *FS {
	fs := &FS{
		EntryBase: plugin.NewEntry(name),
	}
	fs.executor = executor
	fs.maxdepth = maxdepth
	fs.SetTTLOf(plugin.ListOp, ListTTL)
	return fs
}

// ChildSchemas returns the FS entry's child schema
func (d *FS) ChildSchemas() []*plugin.EntrySchema {
	return ChildSchemas()
}

// Schema returns the FS entry's schema
func (d *FS) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(d, "fs").
		SetDescription(fsDescription).
		IsSingleton()
}

// List creates a hierarchy of the filesystem of an Execable resource (the executor).
func (d *FS) List(ctx context.Context) ([]plugin.Entry, error) {
	return List(ctx, d)
}

type nonZeroError struct {
	cmdline  []string
	output   string
	exitcode int
}

func (e nonZeroError) Error() string {
	return fmt.Sprintf("Exec exited non-zero [%v] running %v: %v", e.exitcode, strings.Join(e.cmdline, " "), e.output)
}

func exec(ctx context.Context, executor plugin.Execable, cmdline []string) (*bytes.Buffer, error) {
	// Use Elevate because it's common to login to systems as a non-root user and sudo.
	// Use Tty if running Wash interactively so we get a reflection of the system consistent with
	// being logged in as a user. `ls` will report different file types based on whether you're using
	// it interactively, see character device vs named pipe on /dev/stderr as an example. This also
	// ensures we cleanup correctly when the context is cancelled.
	opts := plugin.ExecOptions{Elevate: true, Tty: plugin.IsInteractive()}
	cmd, err := plugin.Exec(ctx, executor, cmdline[0], cmdline[1:], opts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	var errs []error
	for chunk := range cmd.OutputCh() {
		if chunk.Err != nil {
			errs = append(errs, chunk.Err)
		} else {
			activity.Record(ctx, "%v: %v", chunk.StreamID, chunk.Data)
			if chunk.StreamID == plugin.Stdout {
				fmt.Fprint(&buf, chunk.Data)
			}
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("exec errored: %v", errs)
	}

	exitcode, err := cmd.ExitCode()
	if err != nil {
		return nil, err
	} else if exitcode != 0 {
		// Can happen due to permission denied. Leave handling up to the caller.
		return &buf, nonZeroError{cmdline: cmdline, output: buf.String(), exitcode: exitcode}
	}
	return &buf, nil
}

// VolumeList satisfies the Interface required by List to enumerate files.
func (d *FS) VolumeList(ctx context.Context, path string) (DirMap, error) {
	cmdline := StatCmd(path, d.maxdepth)
	activity.Record(ctx, "Running %v on %v", cmdline, plugin.ID(d.executor))
	buf, err := exec(ctx, d.executor, cmdline)
	if _, ok := err.(nonZeroError); ok {
		// May not have access to some files, but list the rest.
		activity.Record(ctx, "%v running %v, attempting to parse output", err, cmdline)
	} else if err != nil {
		activity.Record(ctx, "Exec error running %v in VolumeList: %v", cmdline, err)
		return nil, err
	}
	activity.Record(ctx, "VolumeList complete")
	// Always returns results normalized to the base.
	return StatParseAll(buf, RootPath, path, d.maxdepth)
}

// VolumeOpen satisfies the Interface required by List to read file contents.
func (d *FS) VolumeOpen(ctx context.Context, path string) (plugin.SizedReader, error) {
	activity.Record(ctx, "Reading %v on %v", path, plugin.ID(d.executor))
	buf, err := exec(ctx, d.executor, []string{"cat", path})
	if err != nil {
		activity.Record(ctx, "Exec error running 'cat %v' in VolumeOpen: %v", path, err)
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}

// VolumeStream satisfies the Interface required by List to stream file contents.
func (d *FS) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	activity.Record(ctx, "Streaming %v on %v", path, plugin.ID(d.executor))
	execOpts := plugin.ExecOptions{Elevate: true, Tty: true}
	cmd, err := plugin.Exec(ctx, d.executor, "tail", []string{"-f", path}, execOpts)
	if err != nil {
		activity.Record(ctx, "Exec error in VolumeRead: %v", err)
		return nil, err
	}

	r, w := io.Pipe()
	go func() {
		// Exec uses context; if it's canceled, the OutputCh will close. Close the writer.
		var errs []error
		for chunk := range cmd.OutputCh() {
			if chunk.Err != nil {
				activity.Record(ctx, "Error on exec: %v", chunk.Err)
				errs = append(errs, chunk.Err)
				continue
			}

			activity.Record(ctx, "%v: %v", chunk.StreamID, chunk.Data)
			if len(errs) == 0 {
				if _, err := w.Write([]byte(chunk.Data)); err != nil {
					activity.Record(ctx, "Error copying exec result: %v", err)
					errs = append(errs, err)
				}
			}
		}

		if len(errs) > 0 {
			err = w.CloseWithError(fmt.Errorf("Multiple errors from exec output: %v", errs))
		} else {
			err = w.Close()
		}
		activity.Record(ctx, "Closing write pipe: %v", err)
	}()
	return r, nil
}

// VolumeDelete satisfies the Interface required by Delete to delete volume nodes.
func (d *FS) VolumeDelete(ctx context.Context, path string) (bool, error) {
	activity.Record(ctx, "Deleting %v on %v", path, plugin.ID(d.executor))
	_, err := exec(ctx, d.executor, []string{"rm", "-rf", path})
	if err != nil {
		activity.Record(ctx, "Exec error running 'rm -rf %v' in VolumeDelete: %v", path, err)
		return false, err
	}
	return true, nil
}

const fsDescription = `
This represents the root directory of a container/VM. It lets you navigate
and interact with that container/VM's filesystem as if you were logged into
it. Thus, you're able to do things like 'cat'/'tail' that container/VM's files
(or even multiple files spread out across multiple containers/VMs).

Note that Wash will exec a command on the container/VM whenever it invokes a
List/Read/Stream action on a directory/file, and the action's result is not
currently cached. For List, that command is 'find -exec stat'. For Read, that
command is 'cat'. For Stream, that command is 'tail -f'.
`

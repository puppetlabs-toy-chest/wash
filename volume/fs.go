package volume

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// FS presents a view of the filesystem of an Execable resource.
type FS struct {
	plugin.EntryBase
	executor plugin.Execable
}

// NewFS creates a new FS entry with the given name, using the supplied executor to satisfy volume
// operations.
func NewFS(name string, executor plugin.Execable) *FS {
	fs := &FS{EntryBase: plugin.NewEntry(name), executor: executor}
	// Caching handled in List.
	fs.DisableCachingFor(plugin.ListOp)
	return fs
}

// List will attempt to list the filesystem of an Execable resource (the executor). It will list
// a directory tree based on supplied configuration (defaulting to `/var/log` if not specified).
func (d *FS) List(ctx context.Context) ([]plugin.Entry, error) {
	return List(ctx, d, "")
}

var errNonZero = fmt.Errorf("Exec exited non-zero")

func exec(ctx context.Context, executor plugin.Execable, cmdline []string) (*bytes.Buffer, error) {
	cmd, err := executor.Exec(ctx, cmdline[0], cmdline[1:], plugin.ExecOptions{Elevate: true})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	var errs []error
	for chunk := range cmd.OutputCh {
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

	exitcode, err := cmd.ExitCodeCB()
	if err != nil {
		return nil, err
	} else if exitcode != 0 {
		// Can happen due to permission denied. Leave handling up to the caller.
		return &buf, errNonZero
	}
	return &buf, nil
}

// VolumeList satisfies the Interface required by List to enumerate files.
func (d *FS) VolumeList(ctx context.Context) (DirMap, error) {
	cmdline := StatCmd("/var/log")
	activity.Record(ctx, "Running %v on %v", cmdline, plugin.ID(d.executor))
	buf, err := exec(ctx, d.executor, cmdline)
	if err == errNonZero {
		// May not have access to some files, but list the rest.
		activity.Record(ctx, "%v running %v, attempting to parse output", err, cmdline)
	} else if err != nil {
		activity.Record(ctx, "Exec error running %v in VolumeList: %v", cmdline, err)
		return nil, err
	}
	activity.Record(ctx, "VolumeList complete")
	return StatParseAll(buf, "")
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
	cmd, err := d.executor.Exec(ctx, "tail", []string{"-f", path}, execOpts)
	if err != nil {
		activity.Record(ctx, "Exec error in VolumeRead: %v", err)
		return nil, err
	}

	// Setup context cancellation handling
	if cmd.StopFunc != nil {
		go func() {
			<-ctx.Done()
			cmd.StopFunc()
		}()
	}

	r, w := io.Pipe()
	go func() {
		// Exec uses context; if it's canceled, the OutputCh will close. Close the writer.
		var errs []error
		for chunk := range cmd.OutputCh {
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

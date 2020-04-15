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
func NewFS(ctx context.Context, name string, executor plugin.Execable, maxdepth int) *FS {
	fs := &FS{
		EntryBase: plugin.NewEntry(name),
	}
	fs.executor = executor
	fs.maxdepth = maxdepth
	fs.SetTTLOf(plugin.ListOp, ListTTL)

	if _, err := plugin.List(ctx, fs); err != nil {
		fs.MarkInaccessible(ctx, err)
	}

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
	stderr   string
	exitcode int
}

func (e nonZeroError) Error() string {
	return fmt.Sprintf("Exec exited non-zero [%v] running %v:\n%v", e.exitcode, strings.Join(e.cmdline, " "), e.stderr)
}

func exec(ctx context.Context, executor plugin.Execable, cmdline []string, tty bool) (*bytes.Buffer, error) {
	// Use Elevate because it's common to login to systems as a non-root user and sudo.
	opts := plugin.ExecOptions{Elevate: true, Tty: tty}
	cmd, err := plugin.Exec(ctx, executor, cmdline[0], cmdline[1:], opts)
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	var errs []error
	for chunk := range cmd.OutputCh() {
		if chunk.Err != nil {
			errs = append(errs, chunk.Err)
		} else {
			activity.Record(ctx, "%v: %v", chunk.StreamID, chunk.Data)
			switch chunk.StreamID {
			case plugin.Stdout:
				fmt.Fprint(&stdout, chunk.Data)
			case plugin.Stderr:
				fmt.Fprint(&stderr, chunk.Data)
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
		return &stdout, nonZeroError{cmdline: cmdline, stderr: strings.TrimSpace(stderr.String()), exitcode: exitcode}
	}
	return &stdout, nil
}

// VolumeList satisfies the Interface required by List to enumerate files.
func (d *FS) VolumeList(ctx context.Context, path string) (DirMap, error) {
	cmdline := d.selectShellCommand(StatCmdPOSIX(path, d.maxdepth), StatCmdPowershell(path, d.maxdepth))
	activity.Record(ctx, "Running %v on %v", cmdline, plugin.ID(d.executor))

	// Use Tty if running Wash interactively so we get a reflection of the system consistent with
	// being logged in as a user. `ls` will report different file types based on whether you're using
	// it interactively, see character device vs named pipe on /dev/stderr as an example.
	buf, err := exec(ctx, d.executor, cmdline, plugin.IsInteractive())
	if nzerr, ok := err.(nonZeroError); ok {
		// Some messages are considered normal, such as when stat fails because a file no longer exists
		// as part of `find ... -exec stat`. We ignore these errors, but if we see any other errors
		// we fail VolumeList.
		var normalError func(string) bool
		switch d.loginShell() {
		case plugin.POSIXShell:
			normalError = NormalErrorPOSIX
		case plugin.PowerShell:
			normalError = NormalErrorPowerShell
		}

		for _, line := range strings.Split(nzerr.stderr, "\n") {
			if text := strings.TrimSpace(line); text != "" && !normalError(text) {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	}
	activity.Record(ctx, "VolumeList complete")

	// Always returns results normalized to the base.
	switch d.loginShell() {
	case plugin.POSIXShell:
		return ParseStatPOSIX(buf, RootPath, path, d.maxdepth)
	case plugin.PowerShell:
		return ParseStatPowershell(buf, RootPath, path, d.maxdepth)
	default:
		panic("unknown shell")
	}
}

// VolumeRead satisfies the Interface required by List to read file contents.
func (d *FS) VolumeRead(ctx context.Context, path string) ([]byte, error) {
	activity.Record(ctx, "Reading %v on %v", path, plugin.ID(d.executor))
	command := d.selectShellCommand([]string{"cat", path}, []string{"Get-Content '" + path + "'"})

	// Don't use Tty when outputting file content because it may convert LF to CRLF.
	buf, err := exec(ctx, d.executor, command, false)
	if err != nil {
		activity.Record(ctx, "Exec error running %+v in VolumeOpen: %v", command, err)
		return nil, err
	}
	return buf.Bytes(), nil
}

// VolumeStream satisfies the Interface required by List to stream file contents.
func (d *FS) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	activity.Record(ctx, "Streaming %v on %v", path, plugin.ID(d.executor))
	command := d.selectShellCommand(
		[]string{"tail", "-f", path},
		[]string{"Get-Content -Wait -Tail 10 '" + path + "'"},
	)

	execOpts := plugin.ExecOptions{Elevate: true, Tty: true}
	cmd, err := plugin.Exec(ctx, d.executor, command[0], command[1:], execOpts)
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
	command := d.selectShellCommand(
		[]string{"rm", "-rf", path},
		[]string{"Remove-Item -Recurse -Force '" + path + "'"},
	)

	// Skip tty because we don't need it, we ignore the output.
	_, err := exec(ctx, d.executor, command, false)
	if err != nil {
		activity.Record(ctx, "Exec error running 'rm -rf %v' in VolumeDelete: %v", path, err)
		return false, err
	}
	return true, nil
}

// Selects between a posix and powershell command based on the entry's login shell.
// Note that powershell commands are often a single string because they represent a PowerShell
// expression, and it's easier to pass that as a string than try to correctly escape it as
// multiple tokens.
func (d *FS) selectShellCommand(posix []string, power []string) []string {
	switch d.loginShell() {
	case plugin.POSIXShell:
		return posix
	case plugin.PowerShell:
		return power
	default:
		panic("unknown shell")
	}
}

func (d *FS) loginShell() plugin.Shell {
	attr := plugin.Attributes(d.executor)
	if shell := attr.OS().LoginShell; attr.HasOS() && shell != plugin.UnknownShell {
		return shell
	}
	// Fallback to posix as a default
	return plugin.POSIXShell
}

// VolumeExec executes cmd in the directory at path.
func (d *FS) VolumeExec(ctx context.Context, path string, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	opts.WorkingDir = path
	return d.executor.Exec(ctx, cmd, args, opts)
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

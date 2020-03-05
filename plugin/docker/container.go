package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	vol "github.com/puppetlabs/wash/volume"
)

type container struct {
	plugin.EntryBase
	id     string
	client *client.Client
}

func newContainer(inst types.Container, client *client.Client) *container {
	name := inst.ID
	if len(inst.Names) > 0 {
		// The docker API prefixes all names with '/', so remove that.
		// We don't append ID because names must currently be unique in the docker runtime.
		// It's also not clear why 'Names' is an array; `/containers/{id}/json` returns a single
		// Name field while '/containers/json' uses a Names array for each instance. In practice
		// it appears to always be a single name, so take the first as the canonical name.
		name = strings.TrimPrefix(inst.Names[0], "/")
	}
	cont := &container{
		EntryBase: plugin.NewEntry(name),
	}
	cont.id = inst.ID
	cont.client = client

	startTime := time.Unix(inst.Created, 0)
	cont.
		SetPartialMetadata(inst).
		Attributes().
		SetCrtime(startTime).
		SetMtime(startTime).
		SetCtime(startTime).
		SetAtime(startTime)

	return cont
}

func (c *container) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	// Use raw to also get the container size.
	_, raw, err := c.client.ContainerInspectWithRaw(ctx, c.id, true)
	if err != nil {
		return nil, err
	}

	return plugin.ToJSONObject(raw), nil
}

func (c *container) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(c, "container").
		SetPartialMetadataSchema(types.Container{}).
		SetMetadataSchema(types.ContainerJSON{}).
		AddSignal("start", "Starts the container. Equivalent to 'docker start <container>'").
		AddSignal("stop", "Stops the container. Equivalent to 'docker stop <container>'").
		AddSignal("pause", "Suspends all processes in the container. Equivalent to 'docker pause <container>'").
		AddSignal("resume", "Un-suspends all processes in the container. Equivalent to 'docker unpause <container>'").
		AddSignal("restart", "Restarts the container. Equivalent to 'docker restart <container>'").
		AddSignalGroup("linux", `\Asig.+`, "Consists of all the supported Linux signals like SIGHUP, SIGKILL. Equivalent to\n'docker kill <container> --signal <signal>'")
}

func (c *container) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&containerLogFile{}).Schema(),
		(&plugin.MetadataJSONFile{}).Schema(),
		(&vol.FS{}).Schema(),
	}
}

func (c *container) List(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: May be worth creating a helper that makes it easy to create
	// read-only files. Lots of shared code between these two.
	cm, err := plugin.NewMetadataJSONFile(ctx, c)
	if err != nil {
		return nil, err
	}
	clf := newContainerLogFile(c)

	// Include a view of the remote filesystem using volume.FS. Use a small maxdepth because
	// VMs can have lots of files and Exec is fast.
	return []plugin.Entry{clf, cm, vol.NewFS(ctx, "fs", c, 3)}, nil
}

func (c *container) Delete(ctx context.Context) (bool, error) {
	err := c.client.ContainerRemove(ctx, c.id, types.ContainerRemoveOptions{
		Force: true,
	})
	return true, err
}

func (c *container) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	command := append([]string{cmd}, args...)
	activity.Record(ctx, "Exec %v on %v", command, c.Name())

	cfg := types.ExecConfig{Cmd: command, AttachStdout: true, AttachStderr: true, Tty: opts.Tty}
	if opts.Stdin != nil || opts.Tty {
		cfg.AttachStdin = true
	}
	created, err := c.client.ContainerExecCreate(ctx, c.id, cfg)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.ContainerExecAttach(ctx, created.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}

	// If stdin is supplied, asynchronously copy it to container exec input.
	if opts.Stdin != nil {
		go func() {
			_, writeErr := io.Copy(resp.Conn, opts.Stdin)
			// If using a Tty, wait until done reading in case we need to send Ctrl-C; when a Tty is
			// allocated commands expect user input and will respond to control signals. Otherwise close
			// input now to ensure commands that depend on EOF execute correctly.
			if !opts.Tty {
				respErr := resp.CloseWrite()
				activity.Record(ctx, "Closed execution input stream for %v: %v, %v", c.Name(), writeErr, respErr)
			}
		}()
	}

	execCmd := plugin.NewExecCommand(ctx)
	execCmd.SetStopFunc(func() {
		// Close the response on cancellation. Copying will block until there's more to read from the
		// exec output. For an action with no more output it may never return.
		if opts.Tty {
			// If resp.Conn is still open, send Ctrl-C over resp.Conn before closing it.
			_, err := resp.Conn.Write([]byte{0x03})
			activity.Record(ctx, "Sent ETX on context termination: %v", err)
		}
		resp.Close()
	})
	// Asynchronously copy container exec output, then fetch the exit code once
	// the copy's finished.
	go func() {
		_, err := stdcopy.StdCopy(execCmd.Stdout(), execCmd.Stderr(), resp.Reader)
		activity.Record(ctx, "Exec on %v complete: %v", c.Name(), err)
		execCmd.CloseStreamsWithError(err)
		resp.Close()

		// Command's finished. Now send the exit code.
		resp, err := c.client.ContainerExecInspect(ctx, created.ID)
		if err != nil {
			execCmd.SetExitCodeErr(err)
			return
		}
		if resp.Running {
			execCmd.SetExitCodeErr(fmt.Errorf("the command was marked as 'Running' even though the output streams reached EOF"))
			return
		}
		activity.Record(ctx, "Exec on %v exited %v", c.Name(), resp.ExitCode)
		execCmd.SetExitCode(resp.ExitCode)
	}()
	return execCmd, nil
}

func (c *container) Signal(ctx context.Context, signal string) error {
	var err error
	switch signal {
	case "start":
		err = c.client.ContainerStart(ctx, c.id, types.ContainerStartOptions{})
	case "stop":
		err = c.client.ContainerStop(ctx, c.id, nil)
	case "pause":
		err = c.client.ContainerPause(ctx, c.id)
	case "resume":
		err = c.client.ContainerUnpause(ctx, c.id)
	case "restart":
		err = c.client.ContainerRestart(ctx, c.id, nil)
	default:
		// linux signal
		err = c.client.ContainerKill(ctx, c.id, signal)
	}
	return err
}

package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
)

type container struct {
	plugin.EntryBase
	client *client.Client
}

// Metadata
func (c *container) Metadata(ctx context.Context) (plugin.MetadataMap, error) {
	// Use raw to also get the container size.
	_, raw, err := c.client.ContainerInspectWithRaw(ctx, c.Name(), true)
	if err != nil {
		return nil, err
	}

	return plugin.ToMetadata(raw), nil
}

func (c *container) List(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{
		&containerMetadata{plugin.NewEntry("metadata.json"), c},
		&containerLogFile{plugin.NewEntry("log"), c.Name(), c.client},
	}, nil
}

func (c *container) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecResult, error) {
	execResult := plugin.ExecResult{}

	command := append([]string{cmd}, args...)
	cfg := types.ExecConfig{Cmd: command, AttachStdout: true, AttachStderr: true}
	if opts.Stdin != nil {
		cfg.AttachStdin = true
	}
	created, err := c.client.ContainerExecCreate(ctx, c.Name(), cfg)
	if err != nil {
		return execResult, err
	}

	resp, err := c.client.ContainerExecAttach(ctx, created.ID, types.ExecStartCheck{})
	if err != nil {
		return execResult, err
	}

	outputCh, stdout, stderr := plugin.CreateExecOutputStreams(ctx)
	go func() {
		defer resp.Close()

		_, err := stdcopy.StdCopy(stdout, stderr, resp.Reader)
		journal.Record(ctx, "Exec on %v complete: %v", c.Name(), err)
		stdout.CloseWithError(err)
		stderr.CloseWithError(err)
	}()

	var writeErr error
	if opts.Stdin != nil {
		go func() {
			_, writeErr = io.Copy(resp.Conn, opts.Stdin)
			journal.Record(ctx, "Closed execution response stream for %v: %v", c.Name(), writeErr)
		}()
	}

	execResult.OutputCh = outputCh
	execResult.ExitCodeCB = func() (int, error) {
		if writeErr != nil {
			return 0, err
		}

		resp, err := c.client.ContainerExecInspect(ctx, created.ID)
		if err != nil {
			return 0, err
		}

		if resp.Running {
			return 0, fmt.Errorf("the command was marked as 'Running' even though the output streams reached EOF")
		}

		journal.Record(ctx, "Exec on %v exited %v", c.Name(), resp.ExitCode)
		return resp.ExitCode, nil
	}

	return execResult, nil
}

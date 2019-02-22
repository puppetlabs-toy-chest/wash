package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
)

type container struct {
	plugin.EntryBase
	client    *client.Client
	startTime time.Time
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

// Attr
func (c *container) Attr() plugin.Attributes {
	return plugin.Attributes{
		Ctime: c.startTime,
		Mtime: c.startTime,
		Atime: c.startTime,
	}
}

func (c *container) LS(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{
		&containerMetadata{plugin.NewEntry("metadata.json"), c},
		&containerLogFile{plugin.NewEntry("log"), c.Name(), c.client},
	}, nil
}

func (c *container) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (*plugin.ExecResult, error) {
	command := append([]string{cmd}, args...)
	created, err := c.client.ContainerExecCreate(
		ctx,
		c.Name(),
		types.ExecConfig{Cmd: command, AttachStdout: true, AttachStderr: true},
	)

	if err != nil {
		return nil, err
	}

	resp, err := c.client.ContainerExecAttach(ctx, created.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}

	err = c.client.ContainerExecStart(ctx, created.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}

	// TODO: Problem with separating stdout and stderr via. HasStderr are concurrency issues
	execResult := &plugin.ExecResult{}

	// TODO: Add datastore.SyncPipe(). Use that to return these objects
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	go func() {
		defer stdoutW.Close()
		defer stderrW.Close()

		// TODO: Figure out how to handle the error here. Should we let the caller
		// pass-us back a channel indicating the status of the write operations?
		if _, err := stdcopy.StdCopy(stdoutW, stderrW, resp.Reader); err != nil {
			log.Debugf("Errored while writing output: %v", err)
		}
	}()

	execResult.Stdout = stdoutR
	execResult.Stderr = stderrR
	execResult.ExitCodeCB = func() (int, error) {
		resp, err := c.client.ContainerExecInspect(ctx, created.ID)
		if err != nil {
			return 0, err
		}

		if resp.Running {
			return 0, fmt.Errorf("The command was marked as 'Running' even though the OutputStream reached EOF")
		}

		return resp.ExitCode, nil
	}

	return execResult, nil
}

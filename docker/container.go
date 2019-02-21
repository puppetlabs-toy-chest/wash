package docker

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
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

type execOutput struct {
	io.Reader
	hr *types.HijackedResponse
}

func (r *execOutput) Close() error {
	r.hr.Close()
	return nil
}

func (c *container) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (io.Reader, error) {
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

	// NOTE: Need this in order to get the right exit code (to start the actual
	// exec process)
	err = c.client.ContainerExecStart(ctx, created.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}

	_, err = c.client.ContainerExecInspect(ctx, created.ID)
	if err != nil {
		return nil, err
	}

	return &execOutput{resp.Reader, &resp}, nil
}

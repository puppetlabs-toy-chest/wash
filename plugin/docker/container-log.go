package docker

import (
	"bytes"
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

type containerLogFile struct {
	plugin.EntryBase
	containerName string
	client        *client.Client
}

func (clf *containerLogFile) isTty(ctx context.Context) bool {
	meta, err := clf.client.ContainerInspect(ctx, clf.containerName)
	if err == nil {
		return meta.Config.Tty
	}
	activity.Record(ctx, "Error reading info for container %v: %v", clf.containerName, err)
	// Assume true so we don't try to process output if there was an error.
	return true
}

func (clf *containerLogFile) Open(ctx context.Context) (plugin.SizedReader, error) {
	opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	rdr, err := clf.client.ContainerLogs(ctx, clf.containerName, opts)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	var n int64
	if clf.isTty(ctx) {
		if n, err = buf.ReadFrom(rdr); err != nil {
			return nil, err
		}
	} else {
		// Write stdout and stderr to the same buffer.
		if n, err = stdcopy.StdCopy(&buf, &buf, rdr); err != nil {
			return nil, err
		}
	}
	activity.Record(ctx, "Read %v bytes of %v log", n, clf.containerName)

	return bytes.NewReader(buf.Bytes()), nil
}

func (clf *containerLogFile) Stream(ctx context.Context) (io.ReadCloser, error) {
	opts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "10"}
	rdr, err := clf.client.ContainerLogs(ctx, clf.containerName, opts)
	if err != nil {
		return nil, err
	}

	if clf.isTty(ctx) {
		return rdr, nil
	}

	r, w := io.Pipe()
	go func() {
		if _, err = stdcopy.StdCopy(w, w, rdr); err != nil {
			activity.Record(ctx, "Errored reading container %v: %v", clf.containerName, err)
		}
		activity.Record(ctx, "Closing write pipe: %v", w.Close())
	}()
	return r, nil
}

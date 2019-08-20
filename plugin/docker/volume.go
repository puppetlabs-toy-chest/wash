package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	docontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	vol "github.com/puppetlabs/wash/volume"
)

type volume struct {
	plugin.EntryBase
	client *client.Client
}

const mountpoint = "/mnt"

func newVolume(c *client.Client, v *types.Volume) (*volume, error) {
	startTime, err := time.Parse(time.RFC3339, v.CreatedAt)
	if err != nil {
		return nil, err
	}

	vol := &volume{
		EntryBase: plugin.NewEntry(v.Name),
	}
	vol.client = c
	vol.
		Attributes().
		SetCrtime(startTime).
		SetMtime(startTime).
		SetCtime(startTime).
		SetAtime(startTime).
		SetMeta(v)

	return vol, nil
}

func (v *volume) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(v, "volume").
		SetMetaAttributeSchema(types.Volume{})
}

func (v *volume) ChildSchemas() []*plugin.EntrySchema {
	return vol.ChildSchemas()
}

func (v *volume) List(ctx context.Context) ([]plugin.Entry, error) {
	return vol.List(ctx, v)
}

// Create a container that mounts a volume to a default mountpoint and runs a command.
func (v *volume) createContainer(ctx context.Context, cmd []string) (string, error) {
	// Use tty to avoid messing with the extra log formatting.
	cfg := docontainer.Config{Image: "busybox", Cmd: cmd, Tty: true}
	mounts := []mount.Mount{{
		Type:     mount.TypeVolume,
		Source:   v.Name(),
		Target:   mountpoint,
		ReadOnly: true,
	}}
	hostcfg := docontainer.HostConfig{Mounts: mounts}
	netcfg := network.NetworkingConfig{}
	created, err := v.client.ContainerCreate(ctx, &cfg, &hostcfg, &netcfg, "")
	if err != nil {
		return "", err
	}
	for _, warn := range created.Warnings {
		activity.Record(ctx, "Warning creating %v: %v", created.ID, warn)
	}
	return created.ID, nil
}

func (v *volume) VolumeList(ctx context.Context, path string) (vol.DirMap, error) {
	// Use a larger maxdepth because volumes have relatively few files and VolumeList is slow.
	maxdepth := 10

	// Create a container that mounts a volume and inspects it. Run it and capture the output.
	cid, err := v.createContainer(ctx, vol.StatCmd(mountpoint+path, maxdepth))
	if err != nil {
		return nil, err
	}
	defer func() {
		err := v.client.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{})
		activity.Record(ctx, "Deleted container %v: %v", cid, err)
	}()

	activity.Record(ctx, "Starting container %v", cid)
	if err := v.client.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	activity.Record(ctx, "Waiting for container %v", cid)
	waitC, errC := v.client.ContainerWait(ctx, cid, docontainer.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-errC:
		return nil, err
	case result := <-waitC:
		statusCode = result.StatusCode
		activity.Record(ctx, "Container %v finished[%v]: %v", cid, result.StatusCode, result.Error)
	}

	opts := types.ContainerLogsOptions{ShowStdout: true}
	if statusCode != 0 {
		opts.ShowStderr = true
	}

	activity.Record(ctx, "Gathering log for %v", cid)
	output, err := v.client.ContainerLogs(ctx, cid, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		activity.Record(ctx, "Closed log for %v: %v", cid, output.Close())
	}()

	if statusCode != 0 {
		bytes, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bytes))
	}

	return vol.StatParseAll(output, mountpoint, path, maxdepth)
}

func (v *volume) VolumeOpen(ctx context.Context, path string) (plugin.SizedReader, error) {
	// Create a container that mounts a volume and waits. Use it to download a file.
	cid, err := v.createContainer(ctx, []string{"sleep", "60"})
	if err != nil {
		return nil, err
	}
	defer func() {
		err := v.client.ContainerRemove(context.Background(), cid, types.ContainerRemoveOptions{})
		activity.Record(ctx, "Deleted temporary container %v: %v", cid, err)
	}()

	activity.Record(ctx, "Starting container %v", cid)
	if err := v.client.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	defer func() {
		err := v.client.ContainerKill(context.Background(), cid, "SIGKILL")
		activity.Record(ctx, "Stopped temporary container %v: %v", cid, err)
	}()

	// Download file, then kill container.
	rdr, _, err := v.client.CopyFromContainer(ctx, cid, mountpoint+path)
	if err != nil {
		return nil, err
	}
	defer func() {
		activity.Record(ctx, "Closed file %v on %v: %v", mountpoint+path, cid, rdr.Close())
	}()

	tarReader := tar.NewReader(rdr)
	// Only expect one file.
	if _, err := tarReader.Next(); err != nil {
		return nil, err
	}
	bits, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(bits), nil
}

func (v *volume) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	// Create a container that mounts a volume and tails a file. Run it and capture the output.
	cid, err := v.createContainer(ctx, []string{"tail", "-f", mountpoint + path})
	if err != nil {
		return nil, err
	}

	// Manually use this in case of errors. On success, the returned Closer will need to call instead.
	delete := func(ct context.Context) {
		err := v.client.ContainerRemove(ct, cid, types.ContainerRemoveOptions{})
		activity.Record(ctx, "Deleted container %v: %v", cid, err)
	}

	activity.Record(ctx, "Starting container %v", cid)
	if err := v.client.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		activity.Record(ctx, "Error starting container %v: %v", cid, err)
		delete(context.Background())
		return nil, err
	}

	// Manually use this in case of errors. On success, the returned Closer will need to call instead.
	killAndDelete := func() {
		ct := context.Background()
		err := v.client.ContainerKill(ct, cid, "SIGKILL")
		activity.Record(ctx, "Stopped temporary container %v: %v", cid, err)
		delete(ct)
	}

	opts := types.ContainerLogsOptions{ShowStdout: true, Follow: true, Tail: "10"}
	activity.Record(ctx, "Streaming log for %v", cid)
	output, err := v.client.ContainerLogs(ctx, cid, opts)
	if err != nil {
		killAndDelete()
		return nil, err
	}

	// Wrap the log output in a ReadCloser that stops and kills the container on Close.
	return plugin.CleanupReader{ReadCloser: output, Cleanup: killAndDelete}, nil
}

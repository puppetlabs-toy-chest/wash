package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	docontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	vol "github.com/puppetlabs/wash/volume"
)

type volume struct {
	plugin.EntryBase
	client *client.Client
}

const mountpoint = "/mnt"

func newVolume(parent plugin.Entry, c *client.Client, v *types.Volume) (*volume, error) {
	startTime, err := time.Parse(time.RFC3339, v.CreatedAt)
	if err != nil {
		return nil, err
	}

	vol := &volume{
		EntryBase: parent.NewEntry(v.Name),
		client:    c,
	}
	vol.SetTTLOf(plugin.ListOp, 60*time.Second)

	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(startTime).
		SetMtime(startTime).
		SetAtime(startTime).
		SetMeta(plugin.ToMeta(v))
	vol.SetAttributes(attr)

	return vol, nil
}

func (v *volume) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	_, raw, err := v.client.VolumeInspectWithRaw(ctx, v.Name())
	if err != nil {
		return nil, err
	}
	return plugin.ToMeta(raw), nil
}

func (v *volume) List(ctx context.Context) ([]plugin.Entry, error) {
	// Create a container that mounts a volume and inspects it. Run it and capture the output.
	cid, err := v.createContainer(ctx, vol.StatCmd(mountpoint))
	if err != nil {
		return nil, err
	}
	defer func() {
		err := v.client.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{})
		journal.Record(ctx, "Deleted container %v: %v", cid, err)
	}()

	journal.Record(ctx, "Starting container %v", cid)
	if err := v.client.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	journal.Record(ctx, "Waiting for container %v", cid)
	waitC, errC := v.client.ContainerWait(ctx, cid, docontainer.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-errC:
		return nil, err
	case result := <-waitC:
		statusCode = result.StatusCode
		journal.Record(ctx, "Container %v finished[%v]: %v", cid, result.StatusCode, result.Error)
	}

	opts := types.ContainerLogsOptions{ShowStdout: true}
	if statusCode != 0 {
		opts.ShowStderr = true
	}

	journal.Record(ctx, "Gathering log for %v", cid)
	output, err := v.client.ContainerLogs(ctx, cid, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		journal.Record(ctx, "Closed log for %v: %v", cid, output.Close())
	}()

	if statusCode != 0 {
		bytes, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bytes))
	}

	dirs, err := vol.StatParseAll(output, mountpoint)
	if err != nil {
		return nil, err
	}

	root := dirs[""]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode().IsDir() {
			entries = append(entries, vol.NewDir(v, name, attr, v.getContentCB(), "/"+name, dirs))
		} else {
			entries = append(entries, vol.NewFile(v, name, attr, v.getContentCB(), "/"+name))
		}
	}
	return entries, nil
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
		journal.Record(ctx, "Warning creating %v: %v", created.ID, warn)
	}
	return created.ID, nil
}

func (v *volume) getContentCB() vol.ContentCB {
	return func(ctx context.Context, path string) (plugin.SizedReader, error) {
		// Create a container that mounts a volume and waits. Use it to download a file.
		cid, err := v.createContainer(ctx, []string{"sleep", "60"})
		if err != nil {
			return nil, err
		}
		defer func() {
			err := v.client.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{})
			journal.Record(ctx, "Deleted temporary container %v: %v", cid, err)
		}()

		journal.Record(ctx, "Starting container %v", cid)
		if err := v.client.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
			return nil, err
		}
		defer func() {
			err := v.client.ContainerKill(ctx, cid, "SIGKILL")
			journal.Record(ctx, "Stopped temporary container %v: %v", cid, err)
		}()

		// Download file, then kill container.
		rdr, _, err := v.client.CopyFromContainer(ctx, cid, mountpoint+path)
		if err != nil {
			return nil, err
		}
		defer func() {
			journal.Record(ctx, "Closed file %v on %v: %v", mountpoint+path, cid, rdr.Close())
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
}

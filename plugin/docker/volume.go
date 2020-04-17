package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	docontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	volpkg "github.com/puppetlabs/wash/volume"
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
	vol.SetTTLOf(plugin.ListOp, volpkg.ListTTL)
	vol.
		SetPartialMetadata(v).
		Attributes().
		SetCrtime(startTime).
		SetMtime(startTime).
		SetCtime(startTime).
		SetAtime(startTime)

	return vol, nil
}

func (v *volume) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(v, "volume").
		SetDescription(volumeDescription).
		SetPartialMetadataSchema(types.Volume{})
}

func (v *volume) ChildSchemas() []*plugin.EntrySchema {
	return volpkg.ChildSchemas()
}

func (v *volume) List(ctx context.Context) ([]plugin.Entry, error) {
	return volpkg.List(ctx, v)
}

func (v *volume) Delete(ctx context.Context) (bool, error) {
	err := v.client.VolumeRemove(ctx, v.Name(), true)
	return true, err
}

// Create a container that mounts a volume to a default mountpoint and runs a command.
// Returns the ID for a running container and a deletion function for cleanup.
func (v *volume) createContainer(ctx context.Context, cmd []string) (string, func(), error) {
	// Use tty to avoid messing with the extra log formatting.
	cfg := docontainer.Config{Image: "busybox", Cmd: cmd, Tty: true}
	mounts := []mount.Mount{{
		Type:   mount.TypeVolume,
		Source: v.Name(),
		Target: mountpoint,
	}}
	hostcfg := docontainer.HostConfig{Mounts: mounts}
	netcfg := network.NetworkingConfig{}
	created, err := v.client.ContainerCreate(ctx, &cfg, &hostcfg, &netcfg, "")
	if err != nil {
		// Pull busybox if create failed because it wasn't found.
		// Taken from https://github.com/docker/cli/blob/v19.03.4/cli/command/container/create.go#L218-L241.
		if !client.IsErrNotFound(err) {
			return "", nil, err
		}

		var pullRdr io.ReadCloser
		if pullRdr, err = v.client.ImagePull(ctx, "busybox:latest", types.ImagePullOptions{}); err != nil {
			return "", nil, err
		}
		defer pullRdr.Close()

		writer := activity.Writer{Context: ctx, Prefix: "Pulling busybox"}
		if _, err := io.Copy(writer, pullRdr); err != nil {
			return "", nil, err
		}

		if created, err = v.client.ContainerCreate(ctx, &cfg, &hostcfg, &netcfg, ""); err != nil {
			return "", nil, err
		}
	}
	for _, warn := range created.Warnings {
		activity.Record(ctx, "Warning creating %v: %v", created.ID, warn)
	}

	cid := created.ID
	remove := func(ctx context.Context) {
		err := v.client.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{})
		activity.Record(ctx, "Deleted container %v: %v", cid, err)
	}

	activity.Record(ctx, "Starting container %v", cid)
	if err := v.client.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		activity.Record(ctx, "Error starting container %v: %v", cid, err)
		// Run in the background so we still cleanup containers if the context was cancelled.
		remove(context.Background())
		return "", nil, err
	}

	cleanup := func() {
		// Use a background context to ensure we stop even if the context was cancelled.
		ctx := context.Background()
		err := v.client.ContainerKill(ctx, cid, "SIGKILL")
		activity.Record(ctx, "Stopped temporary container %v: %v", cid, err)
		remove(ctx)
	}
	return cid, cleanup, nil
}

// Runs cmd in a temporary container. If the exit code is 0, then it returns the cmd's output.
// Otherwise, it wraps the cmd's output in an error object.
func (v *volume) runInTemporaryContainer(ctx context.Context, cmd []string) ([]byte, error) {
	// Create a container that mounts a volume and deletes its file. Run rm -rf on it.
	cid, cleanup, err := v.createContainer(ctx, cmd)
	if err != nil {
		return nil, err
	}
	defer cleanup()

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

	bytes, err := ioutil.ReadAll(output)
	if err != nil {
		return nil, err
	}
	if statusCode != 0 {
		return nil, errors.New(strings.Trim(string(bytes), "\n"))
	}
	return bytes, nil
}

func (v *volume) VolumeList(ctx context.Context, path string) (volpkg.DirMap, error) {
	// Use a larger maxdepth because volumes have relatively few files and VolumeList is slow.
	maxdepth := 10
	output, err := v.runInTemporaryContainer(ctx, volpkg.StatCmdPOSIX(mountpoint+path, maxdepth))
	if err != nil {
		return nil, err
	}
	return volpkg.ParseStatPOSIX(bytes.NewReader(output), mountpoint, path, maxdepth)
}

func (v *volume) VolumeRead(ctx context.Context, path string) ([]byte, error) {
	// Create a container that mounts a volume and waits. Use it to download a file.
	cid, cleanup, err := v.createContainer(ctx, []string{"sleep", "60"})
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Download file, then kill container.
	rdr, _, err := v.client.CopyFromContainer(ctx, cid, mountpoint+path)
	if err != nil {
		return nil, err
	}
	defer func() {
		activity.Record(ctx, "Closed file %v on %v: %v", mountpoint+path, cid, rdr.Close())
	}()

	// Read one file from the archive.
	tarReader := tar.NewReader(rdr)
	if _, err = tarReader.Next(); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(tarReader)
}

func (v *volume) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	// Create a container that mounts a volume and tails a file. Run it and capture the output.
	cid, cleanup, err := v.createContainer(ctx, []string{"tail", "-f", mountpoint + path})
	if err != nil {
		return nil, err
	}

	opts := types.ContainerLogsOptions{ShowStdout: true, Follow: true, Tail: "10"}
	activity.Record(ctx, "Streaming log for %v", cid)
	output, err := v.client.ContainerLogs(ctx, cid, opts)
	if err != nil {
		cleanup()
		return nil, err
	}

	// Wrap the log output in a ReadCloser that stops and kills the container on Close.
	return plugin.CleanupReader{ReadCloser: output, Cleanup: cleanup}, nil
}

func (v *volume) VolumeWrite(ctx context.Context, path string, b []byte, mode os.FileMode) error {
	// Create a container that mounts a volume and waits. Use it to upload a file.
	cid, cleanup, err := v.createContainer(ctx, []string{"sleep", "60"})
	if err != nil {
		return err
	}
	defer cleanup()

	// Create a tar of the file contents and upload it. CopyToContainer requires content as a Reader
	// for a TAR archive.
	dir, file := filepath.Split(path)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	mtime := time.Now()
	hdr := tar.Header{
		Name: file,
		Size: int64(len(b)),
		Mode: int64(mode),
		// Use PAX format to ensure compatibility with non-ASCII filenames.
		Format: tar.FormatPAX,
		// Use of PAX requires we set atime/ctime/mtime. Use now, we just read the file to update it.
		AccessTime: mtime,
		ChangeTime: mtime,
		ModTime:    mtime,
	}

	if err := tw.WriteHeader(&hdr); err != nil {
		return err
	} else if _, err := tw.Write(b); err != nil {
		return err
	} else if err := tw.Close(); err != nil {
		return err
	}

	return v.client.CopyToContainer(ctx, cid, mountpoint+dir, &buf, types.CopyToContainerOptions{})
}

func (v *volume) VolumeDelete(ctx context.Context, path string) (bool, error) {
	_, err := v.runInTemporaryContainer(ctx, []string{"rm", "-rf", mountpoint + path})
	if err != nil {
		return false, err
	}
	return true, nil
}

const volumeDescription = `
This is a Docker volume. We create a temporary Docker container whenever
Wash invokes a currently uncached List/Read/Stream action on it or one of
its children. For List, we run 'find -exec stat' on the container and parse
its output. For Read, we run 'sleep 60' then proceed to download the file
content from the container. For Stream, we run 'tail -f' and pass over its
output.
`

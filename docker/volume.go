package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	docontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

// Designed to be used recursively to list the volume hierarchy.
type volume struct {
	*resourcetype
	name string
	path string
	attr plugin.Attributes
	mux  sync.Mutex
}

func newVolume(cli *resourcetype, name string) *volume {
	return &volume{cli, name, "", plugin.Attributes{}, sync.Mutex{}}
}

func (cli *volume) Find(ctx context.Context, name string) (plugin.Node, error) {
	attrs, err := cli.cachedAttributes(ctx)
	if err != nil {
		return nil, err
	}

	if attr, ok := attrs[name]; ok {
		newvol := &volume{cli.resourcetype, cli.name, cli.path + "/" + name, attr, sync.Mutex{}}
		if attr.Mode.IsDir() {
			return plugin.NewDir(newvol), nil
		}
		return plugin.NewFile(newvol), nil
	}

	return nil, plugin.ENOENT
}

func (cli *volume) List(ctx context.Context) ([]plugin.Node, error) {
	attrs, err := cli.cachedAttributes(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]plugin.Node, 0, len(attrs))
	for entry, attr := range attrs {
		if entry == ".." || entry == "." {
			continue
		}

		newvol := &volume{cli.resourcetype, cli.name, cli.path + "/" + entry, attr, sync.Mutex{}}
		if attr.Mode.IsDir() {
			entries = append(entries, plugin.NewDir(newvol))
		} else {
			entries = append(entries, plugin.NewFile(newvol))
		}
	}
	return entries, nil
}

func (cli *volume) String() string {
	return cli.resourcetype.String() + "/" + cli.name + cli.path
}

func (cli *volume) Name() string {
	if cli.path != "" {
		_, file := path.Split(cli.path)
		return file
	}
	return cli.name
}

func (cli *volume) Attr(ctx context.Context) (*plugin.Attributes, error) {
	if cli.path != "" {
		return &cli.attr, nil
	}
	// Rather than load a pod to get mtime, say it's always changing.
	// Leave it up to the caller whether they need to dig further.
	return &plugin.Attributes{Mtime: time.Now(), Valid: validDuration}, nil
}

func (cli *volume) Xattr(ctx context.Context) (map[string][]byte, error) {
	if cli.path == "" {
		// Return metadata for the volume if it's already loaded.
		key := cli.String()
		if entry, err := cli.cache.Get(key); err != nil {
			log.Printf("Cache miss on %v, skipping lookup", key)
		} else {
			log.Debugf("Cache hit on %v", key)
			return plugin.JSONToJSONMap(entry)
		}
	}
	return map[string][]byte{}, nil
}

// TODO: is it a good idea to mix functions? Are NewDir and NewFile enough to differentiate?
func (cli *volume) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()
	return cli.cachedContent(ctx)
}

const mountpoint = "/mnt"

func (cli *volume) cachedAttributes(ctx context.Context) (map[string]plugin.Attributes, error) {
	key := cli.String() + "/list"
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		var attrs map[string]plugin.Attributes
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&attrs)
		return attrs, err
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Printf("Cache miss on %v", key)

	// Create a container that mounts a volume and inspects it. Run it and capture the output.
	cid, err := cli.createContainer(ctx, plugin.StatCmd(mountpoint+cli.path))
	if err != nil {
		return nil, err
	}
	defer cli.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{})

	log.Debugf("Starting container %v", cid)
	if err := cli.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	log.Debugf("Waiting for container %v", cid)
	waitC, errC := cli.ContainerWait(ctx, cid, docontainer.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-errC:
		return nil, err
	case result := <-waitC:
		statusCode = result.StatusCode
		log.Debugf("Container %v finished[%v]: %v", cid, result.StatusCode, result.Error)
	}

	opts := types.ContainerLogsOptions{ShowStdout: true}
	if statusCode != 0 {
		opts.ShowStderr = true
	}

	log.Debugf("Gathering logs for %v", cid)
	output, err := cli.ContainerLogs(ctx, cid, opts)
	if err != nil {
		return nil, err
	}
	defer output.Close()

	// TODO: pod errors if the mounted volume is empty.
	if statusCode != 0 {
		bytes, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bytes))
	}

	scanner := bufio.NewScanner(output)
	attrs := make(map[string]plugin.Attributes)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			attr, name, err := plugin.StatParse(text)
			if err != nil {
				return nil, err
			}
			if name == ".." || name == "." {
				continue
			}
			attrs[name] = attr
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	cli.updated = time.Now()
	err = datastore.CacheAny(cli.cache, key, attrs)
	return attrs, err
}

func (cli *volume) cachedContent(ctx context.Context) (plugin.IFileBuffer, error) {
	key := cli.String() + "/content"
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		return bytes.NewReader(entry), nil
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Printf("Cache miss on %v", key)

	// Create a container that mounts a volume and waits. Use it to download a file.
	cid, err := cli.createContainer(ctx, []string{"sleep", "60"})
	if err != nil {
		return nil, err
	}
	defer cli.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{})

	log.Debugf("Starting container %v", cid)
	if err := cli.ContainerStart(ctx, cid, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}
	defer cli.ContainerKill(ctx, cid, "SIGKILL")

	// Download file, then kill container.
	rdr, _, err := cli.CopyFromContainer(ctx, cid, mountpoint+cli.path)
	if err != nil {
		return nil, err
	}
	defer rdr.Close()

	tarReader := tar.NewReader(rdr)
	// Only expect one file.
	if _, err := tarReader.Next(); err != nil {
		return nil, err
	}
	bits, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return nil, err
	}

	cli.updated = time.Now()
	cli.cache.Set(key, bits)
	return bytes.NewReader(bits), nil
}

// Create a container that mounts a volume to a default mountpoint and runs a command.
func (cli *volume) createContainer(ctx context.Context, cmd []string) (string, error) {
	// Use tty to avoid messing with the extra log formatting.
	cfg := docontainer.Config{Image: "busybox", Cmd: cmd, Tty: true}
	mounts := []mount.Mount{mount.Mount{
		Type:     mount.TypeVolume,
		Source:   cli.name,
		Target:   mountpoint,
		ReadOnly: true,
	}}
	hostcfg := docontainer.HostConfig{Mounts: mounts}
	netcfg := network.NetworkingConfig{}
	created, err := cli.ContainerCreate(ctx, &cfg, &hostcfg, &netcfg, "")
	if err != nil {
		return "", err
	}
	for _, warn := range created.Warnings {
		log.Debugf("Warning creating %v: %v", created.ID, warn)
	}
	return created.ID, nil
}

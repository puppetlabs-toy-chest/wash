package docker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	docontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

// Designed to be used recursively to list the volume hierarchy.
type volume struct {
	*resourcetype
	name string
	path string
}

func (cli *volume) Find(ctx context.Context, name string) (plugin.Node, error) {
	lines, err := cli.cachedList(ctx)
	if err != nil {
		return nil, err
	}

	for _, line := range lines {
		attr, entry, err := parseStat(line)
		if err != nil {
			return nil, err
		}
		if name != entry {
			continue
		}

		newvol := &volume{cli.resourcetype, cli.name, cli.path + "/" + name}
		if attr.Mode.IsDir() {
			return plugin.NewDir(newvol), nil
		}
		return plugin.NewFile(newvol), nil
	}

	return nil, plugin.ENOENT
}

func (cli *volume) List(ctx context.Context) ([]plugin.Node, error) {
	lines, err := cli.cachedList(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]plugin.Node, 0, len(lines))
	for _, line := range lines {
		attr, name, err := parseStat(line)
		if err != nil {
			return nil, err
		}
		if name == ".." || name == "." {
			continue
		}

		newvol := &volume{cli.resourcetype, cli.name, cli.path + "/" + name}
		if attr.Mode.IsDir() {
			entries = append(entries, plugin.NewDir(newvol))
		} else {
			entries = append(entries, plugin.NewFile(newvol))
		}
	}
	return entries, nil
}

func (cli *volume) String() string {
	return cli.resourcetype.String() + "/" + cli.Name() + cli.path
}

func (cli *volume) Name() string {
	if cli.path != "" {
		_, file := path.Split(cli.path)
		return file
	}
	return cli.name
}

func (cli *volume) Attr(ctx context.Context) (*plugin.Attributes, error) {
	return &plugin.Attributes{}, nil
}

func (cli *volume) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

// TODO: is it a good idea to mix functions? Are NewDir and NewFile enough to differentiate?
func (cli *volume) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	return nil, plugin.ENOTSUP
}

func parseStat(line string) (plugin.Attributes, string, error) {
	var attr plugin.Attributes
	segments := strings.SplitN(line, " ", 6)
	if len(segments) != 6 {
		return attr, "", fmt.Errorf("Stat did not return 6 components: %v", line)
	}

	var err error
	attr.Size, err = strconv.ParseUint(segments[0], 10, 64)
	if err != nil {
		return attr, "", err
	}

	for i, target := range []*time.Time{&attr.Atime, &attr.Mtime, &attr.Ctime} {
		epoch, err := strconv.ParseInt(segments[i+1], 10, 64)
		if err != nil {
			return attr, "", err
		}
		*target = time.Unix(epoch, 0)
	}

	mode, err := strconv.ParseUint(segments[4], 16, 32)
	if err != nil {
		return attr, "", err
	}
	attr.Mode = os.FileMode(mode)

	_, file := path.Split(segments[5])

	return attr, file, nil
}

func statFormat() string {
	// size, atime, mtime, ctime, mode, name
	// %s - Total size, in bytes
	// %X - Time of last access as seconds since Epoch
	// %Y - Time of last data modification as seconds since Epoch
	// %Z - Time of last status change as seconds since Epoch
	// %f - Raw mode in hex
	// %n - File name
	return "%s %X %Y %Z %f %n"
}

func (cli *volume) cachedList(ctx context.Context) ([]string, error) {
	return datastore.CachedStrings(cli.BigCache, cli.String(), func() ([]string, error) {
		// Create a container that mounts a volume and inspects it. Run it and capture the output.
		cmd := strslice.StrSlice{"sh", "-c", "stat -c '" + statFormat() + "' /mnt" + cli.path + "/.* /mnt" + cli.path + "/*"}
		// Use tty to avoid messing with the extra log formatting.
		cfg := docontainer.Config{Image: "busybox", Cmd: cmd, Tty: true}
		mounts := []mount.Mount{mount.Mount{
			Type:     mount.TypeVolume,
			Source:   cli.name,
			Target:   "/mnt",
			ReadOnly: true,
		}}
		hostcfg := docontainer.HostConfig{Mounts: mounts}
		netcfg := network.NetworkingConfig{}
		created, err := cli.ContainerCreate(ctx, &cfg, &hostcfg, &netcfg, "")
		if err != nil {
			return nil, err
		}
		for _, warn := range created.Warnings {
			log.Debugf("Warning creating %v: %v", cli.String(), warn)
		}
		cid := created.ID
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

		if statusCode != 0 {
			bytes, err := ioutil.ReadAll(output)
			if err != nil {
				return nil, err
			}
			return nil, errors.New(string(bytes))
		}

		scanner := bufio.NewScanner(output)
		lines := make([]string, 0)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				lines = append(lines, text)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		log.Printf("Lines: %v", lines)
		cli.updated = time.Now()
		return lines, nil
	})
}

package kubernetes

import (
	"context"
	"encoding/json"
	"path"
	"sync"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Designed to be used recursively to list the volume hierarchy.
type pvc struct {
	*resourcetype
	name string
	ns   string
	path string
	attr plugin.Attributes
	mux  sync.Mutex
}

func newPvc(cli *resourcetype, id string) *pvc {
	name, ns := datastore.SplitCompositeString(id)
	return &pvc{cli, name, ns, "", plugin.Attributes{}, sync.Mutex{}}
}

func (cli *pvc) Find(ctx context.Context, name string) (plugin.Node, error) {
	/*
		attrs, err := cli.cachedAttributes(ctx)
		if err != nil {
			return nil, err
		}

		if attr, ok := attrs[name]; ok {
			newvol := &pvc{cli.resourcetype, cli.name, cli.ns, cli.path + "/" + name, attr, sync.Mutex{}}
			if attr.Mode.IsDir() {
				return plugin.NewDir(newvol), nil
			}
			return plugin.NewFile(newvol), nil
		}
	*/

	return nil, plugin.ENOENT
}

func (cli *pvc) List(ctx context.Context) ([]plugin.Node, error) {
	/*
		attrs, err := cli.cachedAttributes(ctx)
		if err != nil {
			return nil, err
		}

		entries := make([]plugin.Node, 0, len(attrs))
		for entry, attr := range attrs {
			if entry == ".." || entry == "." {
				continue
			}

			newvol := &pvc{cli.resourcetype, cli.name, cli.ns, cli.path + "/" + entry, attr, sync.Mutex{}}
			if attr.Mode.IsDir() {
				entries = append(entries, plugin.NewDir(newvol))
			} else {
				entries = append(entries, plugin.NewFile(newvol))
			}
		}
		return entries, nil
	*/
	return []plugin.Node{}, nil
}

// A unique string describing the pod. Note that the same pod may appear in a specific namespace and 'all'.
// It should use the same identifier in both cases.
func (cli *pvc) String() string {
	return cli.resourcetype.client.Name() + "/" + cli.ns + "/pvc/" + cli.Name()
}

func (cli *pvc) Name() string {
	if cli.path != "" {
		_, file := path.Split(cli.path)
		return file
	}
	return cli.name
}

func (cli *pvc) Attr(ctx context.Context) (*plugin.Attributes, error) {
	if cli.path != "" {
		return &cli.attr, nil
	}
	// Rather than load a pod to get mtime, say it's always changing.
	// Leave it up to the caller whether they need to dig further.
	return &plugin.Attributes{Mtime: time.Now(), Valid: validDuration}, nil
}

func (cli *pvc) Xattr(ctx context.Context) (map[string][]byte, error) {
	if cli.path == "" {
		// Return metadata for the pvc if it's already loaded.
		key := cli.String()
		if entry, err := cli.resourcetype.client.cache.Get(key); err != nil {
			log.Printf("Cache miss on %v, skipping lookup", key)
		} else {
			log.Debugf("Cache hit on %v", key)
			return plugin.JSONToJSONMap(entry)
		}
	}
	return map[string][]byte{}, nil
}

// TODO: is it a good idea to mix functions? Are NewDir and NewFile enough to differentiate?
func (cli *pvc) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()
	//return cli.cachedContent(ctx)
	return nil, plugin.ENOTSUP
}

/*
const mountpoint = "/mnt"

func (cli *pvc) cachedAttributes(ctx context.Context) (map[string]plugin.Attributes, error) {
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

	// Create a container that mounts a pvc and inspects it. Run it and capture the output.
	cid, err := cli.startContainer(ctx, plugin.StatCmd(mountpoint+cli.path))
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

func (cli *pvc) cachedContent(ctx context.Context) (plugin.IFileBuffer, error) {
	key := cli.String() + "/content"
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		return bytes.NewReader(entry), nil
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Printf("Cache miss on %v", key)

	// Create a container that mounts a pvc and waits. Use it to download a file.
	cid, err := cli.startContainer(ctx, []string{"sleep", "60"})
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

// Create a container that mounts a pvc to a default mountpoint and runs a command.
func (cli *pvc) startContainer(ctx context.Context, cmd []string) (string, error) {
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
*/

func (cli *client) cachedPvcs(ctx context.Context, ns string) ([]string, error) {
	return datastore.CachedStrings(cli.cache, cli.Name()+"/pvcs/"+ns, func() ([]string, error) {
		pvcList, err := cli.CoreV1().PersistentVolumeClaims("").List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		allpvcs := make([]string, len(pvcList.Items))
		pvcs := make(map[string][]string)
		for i, pvc := range pvcList.Items {
			allpvcs[i] = datastore.MakeCompositeString(pvc.Name, pvc.Namespace)
			pvcs[pvc.Namespace] = append(pvcs[pvc.Namespace], allpvcs[i])
			// Also cache individual pvc data as JSON for use in xattributes.
			if bits, err := json.Marshal(pvc); err == nil {
				cli.cache.Set(cli.Name()+"/"+pvc.Namespace+"/pvc/"+pvc.Name, bits)
			} else {
				log.Printf("Unable to marshal pvc %v: %v", pvc, err)
			}
		}
		pvcs[allNamespace] = allpvcs

		for name, data := range pvcs {
			// Skip the one we're returning because CachedStrings will encode and store to cache for us.
			if name != ns {
				datastore.CacheAny(cli.cache, cli.Name()+"/pvcs/"+name, data)
			}
		}
		return pvcs[ns], nil
	})
}

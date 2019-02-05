package kubernetes

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"sync"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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

	return nil, plugin.ENOENT
}

func (cli *pvc) List(ctx context.Context) ([]plugin.Node, error) {
	attrs, err := cli.cachedAttributes(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]plugin.Node, 0, len(attrs))
	for entry, attr := range attrs {
		newvol := &pvc{cli.resourcetype, cli.name, cli.ns, cli.path + "/" + entry, attr, sync.Mutex{}}
		if attr.Mode.IsDir() {
			entries = append(entries, plugin.NewDir(newvol))
		} else {
			entries = append(entries, plugin.NewFile(newvol))
		}
	}
	return entries, nil
}

func (cli *pvc) baseID() string {
	return cli.resourcetype.client.Name() + "/" + cli.ns + "/pvc/" + cli.Name()
}

// A unique string describing the pod. Note that the same pvc may appear in a specific namespace and 'all'.
// It should use the same identifier in both cases.
func (cli *pvc) String() string {
	return cli.baseID() + cli.path
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
	pid, err := cli.createPod(plugin.StatCmd(mountpoint))
	if err != nil {
		return nil, err
	}
	podi := cli.CoreV1().Pods(cli.ns)
	defer podi.Delete(pid, &metav1.DeleteOptions{})

	log.Debugf("Waiting for pod %v", pid)
	// Start watching for new events related to the pod we created.
	watchOpts := metav1.ListOptions{FieldSelector: "metadata.name=" + pid}
	watcher, err := podi.Watch(watchOpts)
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	ch := watcher.ResultChan()
	success := true
	for ch != nil {
		select {
		case e, ok := <-ch:
			if !ok {
				return nil, fmt.Errorf("Channel error waiting for pod %v: %v", pid, e)
			}
			switch e.Type {
			case watch.Modified:
				switch e.Object.(*corev1.Pod).Status.Phase {
				case corev1.PodSucceeded:
					ch = nil
				case corev1.PodFailed:
					ch = nil
					success = false
				case corev1.PodUnknown:
					log.Printf("Unknown state for pod %v: %v", pid, e.Object)
				}
			case watch.Error:
				return nil, fmt.Errorf("Pod %v errored: %v", pid, e.Object)
			}
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("Timed out waiting for pod %v", pid)
		}
	}

	log.Debugf("Gathering logs for %v", pid)
	output, err := podi.GetLogs(pid, &corev1.PodLogOptions{}).Stream()
	if err != nil {
		return nil, err
	}
	defer output.Close()

	if !success {
		bytes, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bytes))
	}

	attrs, err := plugin.StatParseAll(output, mountpoint)
	if err != nil {
		return nil, err
	}

	for dir, attrmap := range attrs {
		key := cli.baseID() + dir + "list"
		if err = datastore.CacheAny(cli.cache, key, attrmap); err != nil {
			log.Printf("Failed to cache %v: %v", key, err)
		}
	}
	cli.updated = time.Now()
	return attrs[cli.path+"/"], err
}

/*
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
*/

// Create a container that mounts a pvc to a default mountpoint and runs a command.
func (cli *pvc) createPod(cmd []string) (string, error) {
	podi := cli.CoreV1().Pods(cli.ns)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "wash",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  "busybox",
					Image: "busybox",
					Args:  plugin.StatCmd(mountpoint + cli.path),
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      cli.name,
							MountPath: mountpoint,
							ReadOnly:  true,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: cli.name,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: cli.name,
							ReadOnly:  true,
						},
					},
				},
			},
		},
	}
	created, err := podi.Create(pod)
	if err != nil {
		return "", err
	}
	return created.Name, nil
}

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

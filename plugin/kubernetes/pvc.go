package kubernetes

import (
	"bytes"
	"context"
	"io"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/volume"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type pvc struct {
	plugin.EntryBase
	pvci      typedv1.PersistentVolumeClaimInterface
	podi      typedv1.PodInterface
	client    *k8s.Clientset
	config    *rest.Config
	namespace string
}

func newPVC(pi typedv1.PersistentVolumeClaimInterface, client *k8s.Clientset, config *rest.Config, ns string, p *corev1.PersistentVolumeClaim) *pvc {
	vol := &pvc{
		EntryBase: plugin.NewEntry(p.Name),
	}
	vol.pvci = pi
	vol.podi = client.CoreV1().Pods(ns)
	vol.client = client
	vol.config = config
	vol.namespace = ns

	vol.SetTTLOf(plugin.ListOp, volume.ListTTL)
	vol.
		SetPartialMetadata(p).
		Attributes().
		SetCrtime(p.CreationTimestamp.Time).
		SetMtime(p.CreationTimestamp.Time).
		SetCtime(p.CreationTimestamp.Time).
		SetAtime(p.CreationTimestamp.Time)

	return vol
}

func (v *pvc) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(v, "persistentvolumeclaim").
		SetDescription(pvcDescription).
		SetPartialMetadataSchema(corev1.PersistentVolumeClaim{})
}

func (v *pvc) ChildSchemas() []*plugin.EntrySchema {
	return volume.ChildSchemas()
}

func (v *pvc) List(ctx context.Context) ([]plugin.Entry, error) {
	return volume.List(ctx, v)
}

func (v *pvc) Delete(ctx context.Context) (bool, error) {
	err := v.pvci.Delete(v.Name(), &metav1.DeleteOptions{})
	return true, err
}

type mountInfo struct {
	pod       *corev1.Pod
	container *corev1.Container
	path      string
}

// TODO: return read-write mount if available (fallback to read-only) that mounts the volume
// root (SubPath is empty). If no mounts exist with empty SubPath, try mounting in a new pod
// if it's ReadOnly or ReadWriteMany. If that's not possible, error.
func (v *pvc) getFirstMountingPod() (*corev1.Pod, string, error) {
	nsPods, err := v.podi.List(metav1.ListOptions{})
	if err != nil {
		return nil, "", err
	}

	for _, pod := range nsPods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil && vol.PersistentVolumeClaim.ClaimName == v.Name() {
				// Return the name of the volume tied to this claim so we can match it up to the volume
				// mount in a specific container.
				return &pod, vol.Name, nil
			}
		}
	}
	return nil, "", nil
}

func (v *pvc) getMountInfo(pod *corev1.Pod, volumeName string) mountInfo {
	for _, container := range pod.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.Name == volumeName {
				return mountInfo{
					pod:       pod,
					container: &container,
					path:      mount.MountPath,
				}
			}
		}
	}
	panic("volume is mounted, so a container must use it")
}

// Callback is given a mountpoint where the PVC is mounted. It's also responsible for any cleanup
// related to accessing the container because sometimes use of that container persists beyond the
// lifetime of the inContainer function call.
type containerCb = func(c *containerBase, mountpoint string, cleanup func()) (interface{}, error)

// Execution containerCb in a container that has the current PVC mounted. Creates one if one is
// not currently running.
func (v *pvc) inContainer(ctx context.Context, fn containerCb) (interface{}, error) {
	mountingPod, volumeName, err := v.getFirstMountingPod()
	if err != nil {
		return nil, err
	}

	execContainer := containerBase{client: v.client, config: v.config}
	var mountpoint string
	var cleanup func()
	if mountingPod == nil {
		mountpoint = "/mnt"
		tempPod, err := createContainer(v.podi, v.Name(), mountpoint)
		if err != nil {
			return nil, err
		}
		if err := tempPod.waitOnCreation(ctx); err != nil {
			return nil, err
		}
		execContainer.pod = tempPod.pod
		cleanup = func() {
			activity.Record(ctx, "Deleted temporary pod %v: %v", tempPod, tempPod.delete())
		}
	} else {
		mount := v.getMountInfo(mountingPod, volumeName)
		execContainer.pod = mount.pod
		execContainer.container = mount.container
		mountpoint = mount.path
		cleanup = func() {}
	}
	return fn(&execContainer, mountpoint, cleanup)
}

// A constructor for commands that need a path that allows the specific execution context
// to inject a base path.
type cmdBuilder func(string) []string

func (v *pvc) exec(ctx context.Context, buildCmd cmdBuilder) ([]byte, error) {
	obj, err := v.inContainer(ctx, func(c *containerBase, mountpoint string, cleanup func()) (interface{}, error) {
		defer cleanup()

		cmd := buildCmd(mountpoint)
		activity.Record(ctx, "Executing in %v: %v", c, cmd)

		var stdout, stderr bytes.Buffer
		streamOpts := remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr}
		executor, err := c.newExecutor(ctx, cmd[0], cmd[1:], streamOpts)
		if err != nil {
			return []byte{}, err
		}

		err = executor.Stream()
		activity.Record(ctx, "stdout: %v", stdout.String())
		activity.Record(ctx, "stderr: %v", stderr.String())
		return stdout.Bytes(), err
	})
	return obj.([]byte), err
}

func (v *pvc) VolumeList(ctx context.Context, path string) (volume.DirMap, error) {
	// Use a larger maxdepth because volumes generally have few files.
	maxdepth := 10
	var mountpoint string
	output, err := v.exec(ctx, func(base string) []string {
		mountpoint = base
		return volume.StatCmdPOSIX(base+path, maxdepth)
	})

	if err != nil {
		return nil, err
	}
	return volume.ParseStatPOSIX(bytes.NewReader(output), mountpoint, path, maxdepth)
}

func (v *pvc) VolumeRead(ctx context.Context, path string) ([]byte, error) {
	output, err := v.exec(ctx, func(base string) []string {
		return []string{"cat", base + path}
	})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (v *pvc) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	obj, err := v.inContainer(ctx, func(c *containerBase, mountpoint string, cleanup func()) (interface{}, error) {
		cmd := []string{"tail", "-f", mountpoint + path}
		activity.Record(ctx, "Streaming from %v: %v", c, cmd)

		stdoutR, stdoutW := io.Pipe()
		streamOpts := remotecommand.StreamOptions{Stdout: stdoutW, Tty: true}
		executor, err := c.newExecutor(ctx, cmd[0], cmd[1:], streamOpts)
		if err != nil {
			stdoutR.Close()
			stdoutW.Close()
			cleanup()
			return nil, err
		}

		cleanupExec := executor.AsyncStream(func(error) {})
		return plugin.CleanupReader{ReadCloser: stdoutR, Cleanup: func() {
			// Cleanup execution and cleanup the container on completion.
			cleanupExec()
			cleanup()
		}}, nil
	})
	return obj.(io.ReadCloser), err
}

func (v *pvc) VolumeDelete(ctx context.Context, path string) (bool, error) {
	_, err := v.exec(ctx, func(base string) []string {
		return []string{"rm", "-rf", base + path}
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

const pvcDescription = `
This is a Kubernetes persistent volume claim. We create a temporary Kubernetes
pod whenever Wash invokes a currently uncached List/Read/Stream action on it or
one of its children. For List, we run 'find -exec stat' on the pod and parse its
output. For Read, we run 'cat' and return its output. For Stream, we run 'tail -f'
and stream its output.
`

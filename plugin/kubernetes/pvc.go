package kubernetes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/volume"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type pvc struct {
	plugin.EntryBase
	pvci typedv1.PersistentVolumeClaimInterface
	podi typedv1.PodInterface
}

const mountpoint = "/mnt"

var errPodTerminated = errors.New("Pod terminated unexpectedly")

func pvcTemplate() *pvc {
	vol := &pvc{
		EntryBase: plugin.NewEntry(),
	}
	vol.SetShortType("persistentvolumeclaim")
	return vol
}

func newPVC(pi typedv1.PersistentVolumeClaimInterface, pd typedv1.PodInterface, p *corev1.PersistentVolumeClaim) *pvc {
	vol := pvcTemplate()
	vol.pvci = pi
	vol.podi = pd

	vol.
		SetName(p.Name).
		Attributes().
		SetCtime(p.CreationTimestamp.Time).
		SetMtime(p.CreationTimestamp.Time).
		SetAtime(p.CreationTimestamp.Time).
		SetMeta(p)

	return vol
}

func (v *pvc) ChildSchemas() []plugin.EntrySchema {
	return volume.ChildSchemas()
}

func (v *pvc) List(ctx context.Context) ([]plugin.Entry, error) {
	return volume.List(ctx, v, "")
}

// Create a container that mounts a pvc to a default mountpoint and runs a command.
func (v *pvc) createPod(cmd []string) (string, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "wash",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "busybox",
					Image: "busybox",
					Args:  cmd,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      v.Name(),
							MountPath: mountpoint,
							ReadOnly:  true,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: v.Name(),
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: v.Name(),
							ReadOnly:  true,
						},
					},
				},
			},
		},
	}
	created, err := v.podi.Create(pod)
	if err != nil {
		return "", err
	}
	return created.Name, nil
}

func (v *pvc) waitForPod(ctx context.Context, pid string) error {
	watchOpts := metav1.ListOptions{FieldSelector: "metadata.name=" + pid}
	watcher, err := v.podi.Watch(watchOpts)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	ch := watcher.ResultChan()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return fmt.Errorf("Channel error waiting for pod %v: %v", pid, e)
			}
			switch e.Type {
			case watch.Modified:
				switch e.Object.(*corev1.Pod).Status.Phase {
				case corev1.PodSucceeded:
					return nil
				case corev1.PodFailed:
					return errPodTerminated
				case corev1.PodUnknown:
					activity.Record(ctx, "Unknown state for pod %v: %v", pid, e.Object)
				}
			case watch.Error:
				return fmt.Errorf("Pod %v errored: %v", pid, e.Object)
			}
		case <-time.After(30 * time.Second):
			return fmt.Errorf("Timed out waiting for pod %v", pid)
		}
	}
}

func (v *pvc) VolumeList(ctx context.Context) (volume.DirMap, error) {
	// Create a container that mounts a pvc and inspects it. Run it and capture the output.
	pid, err := v.createPod(volume.StatCmd(mountpoint))
	if err != nil {
		return nil, err
	}
	defer func() {
		activity.Record(ctx, "Deleted temporary pod %v: %v", pid, v.podi.Delete(pid, &metav1.DeleteOptions{}))
	}()

	activity.Record(ctx, "Waiting for pod %v to start", pid)
	// Start watching for new events related to the pod we created.
	if err = v.waitForPod(ctx, pid); err != nil && err != errPodTerminated {
		return nil, err
	}

	activity.Record(ctx, "Gathering log for %v", pid)
	output, lerr := v.podi.GetLogs(pid, &corev1.PodLogOptions{}).Stream()
	if lerr != nil {
		return nil, lerr
	}
	defer func() {
		activity.Record(ctx, "Closed log for %v: %v", pid, output.Close())
	}()

	if err == errPodTerminated {
		bytes, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bytes))
	}

	return volume.StatParseAll(output, mountpoint)
}

func (v *pvc) VolumeOpen(ctx context.Context, path string) (plugin.SizedReader, error) {
	// Create a container that mounts a pvc and output the file.
	pid, err := v.createPod([]string{"cat", mountpoint + path})
	activity.Record(ctx, "Reading from: %v", mountpoint+path)
	if err != nil {
		return nil, err
	}
	defer func() {
		activity.Record(ctx, "Deleted temporary pod %v: %v", pid, v.podi.Delete(pid, &metav1.DeleteOptions{}))
	}()

	activity.Record(ctx, "Waiting for pod %v", pid)
	// Start watching for new events related to the pod we created.
	if err = v.waitForPod(ctx, pid); err != nil && err != errPodTerminated {
		return nil, err
	}
	podErr := err

	activity.Record(ctx, "Gathering log for %v", pid)
	output, err := v.podi.GetLogs(pid, &corev1.PodLogOptions{}).Stream()
	if err != nil {
		return nil, err
	}
	defer func() {
		activity.Record(ctx, "Closed log for %v: %v", pid, output.Close())
	}()

	bits, err := ioutil.ReadAll(output)
	if err != nil {
		return nil, err
	}
	activity.Record(ctx, "Read: %v", bits)

	if podErr == errPodTerminated {
		return nil, errors.New(string(bits))
	}
	return bytes.NewReader(bits), nil
}

func (v *pvc) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	// Create a container that mounts a pvc and tail the file.
	pid, err := v.createPod([]string{"tail", "-f", mountpoint + path})
	activity.Record(ctx, "Streaming from: %v", mountpoint+path)
	if err != nil {
		return nil, err
	}

	// Manually use this in case of errors. On success, the returned Closer will need to call instead.
	delete := func() {
		activity.Record(ctx, "Deleted temporary pod %v: %v", pid, v.podi.Delete(pid, &metav1.DeleteOptions{}))
	}

	activity.Record(ctx, "Waiting for pod %v", pid)
	// Start watching for new events related to the pod we created.
	if err = v.waitForPod(ctx, pid); err != nil && err != errPodTerminated {
		delete()
		return nil, err
	}
	podErr := err

	activity.Record(ctx, "Gathering log for %v", pid)
	output, err := v.podi.GetLogs(pid, &corev1.PodLogOptions{}).Stream()
	if err != nil {
		delete()
		return nil, err
	}

	if podErr == errPodTerminated {
		bits, err := ioutil.ReadAll(output)
		activity.Record(ctx, "Closed log for %v: %v", pid, output.Close())
		delete()
		if err != nil {
			return nil, err
		}
		activity.Record(ctx, "Read: %v", bits)

		return nil, errors.New(string(bits))
	}

	// Wrap the log output in a ReadCloser that stops and kills the container on Close.
	return plugin.CleanupReader{ReadCloser: output, Cleanup: delete}, nil
}

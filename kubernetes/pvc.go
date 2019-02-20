package kubernetes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/puppetlabs/wash/os"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type pvc struct {
	plugin.EntryBase
	pvci      typedv1.PersistentVolumeClaimInterface
	podi      typedv1.PodInterface
	startTime time.Time
}

const mountpoint = "/mnt"

var errPodTerminated = errors.New("Pod terminated unexpectedly")

func newPVC(pi typedv1.PersistentVolumeClaimInterface, pd typedv1.PodInterface, p *corev1.PersistentVolumeClaim) *pvc {
	vol := &pvc{
		EntryBase: plugin.NewEntry(p.Name),
		pvci:      pi,
		podi:      pd,
		startTime: p.CreationTimestamp.Time,
	}
	vol.CacheConfig().SetTTLOf(plugin.LS, 30*time.Second)

	return vol
}

func (v *pvc) Metadata(ctx context.Context) (map[string]interface{}, error) {
	obj, err := v.pvci.Get(v.Name(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return plugin.ToMetadata(obj), nil
}

func (v *pvc) Attr() plugin.Attributes {
	return plugin.Attributes{
		Ctime: v.startTime,
		Mtime: v.startTime,
		Atime: v.startTime,
	}
}

func (v *pvc) LS(ctx context.Context) ([]plugin.Entry, error) {
	// Create a container that mounts a pvc and inspects it. Run it and capture the output.
	pid, err := v.createPod(os.StatCmd(mountpoint))
	if err != nil {
		return nil, err
	}
	defer v.podi.Delete(pid, &metav1.DeleteOptions{})

	log.Debugf("Waiting for pod %v", pid)
	// Start watching for new events related to the pod we created.
	if err = v.waitForPod(pid); err != nil && err != errPodTerminated {
		return nil, err
	}

	log.Debugf("Gathering logs for %v", pid)
	output, lerr := v.podi.GetLogs(pid, &corev1.PodLogOptions{}).Stream()
	if lerr != nil {
		return nil, lerr
	}
	defer output.Close()

	if err == errPodTerminated {
		bytes, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(bytes))
	}

	dirs, err := os.StatParseAll(output, mountpoint)
	if err != nil {
		return nil, err
	}

	root := dirs[""]
	entries := make([]plugin.Entry, 0, len(root))
	for name, attr := range root {
		if attr.Mode.IsDir() {
			entries = append(entries, os.NewVolumeDir(name, attr, v.getContentCB(), "/"+name, dirs))
		} else {
			entries = append(entries, os.NewVolumeFile(name, attr, v.getContentCB(), "/"+name))
		}
	}
	return entries, nil
}

// Create a container that mounts a pvc to a default mountpoint and runs a command.
func (v *pvc) createPod(cmd []string) (string, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "wash",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  "busybox",
					Image: "busybox",
					Args:  cmd,
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      v.Name(),
							MountPath: mountpoint,
							ReadOnly:  true,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				corev1.Volume{
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

func (v *pvc) waitForPod(pid string) error {
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
					log.Infof("Unknown state for pod %v: %v", pid, e.Object)
				}
			case watch.Error:
				return fmt.Errorf("Pod %v errored: %v", pid, e.Object)
			}
		case <-time.After(30 * time.Second):
			return fmt.Errorf("Timed out waiting for pod %v", pid)
		}
	}
}

func (v *pvc) getContentCB() os.ContentCB {
	return func(ctx context.Context, path string) (plugin.SizedReader, error) {
		// Create a container that mounts a pvc and waits. Use it to download a file.
		pid, err := v.createPod([]string{"cat", mountpoint + path})
		log.Debugf("Reading from: %v", mountpoint+path)
		if err != nil {
			return nil, err
		}
		defer v.podi.Delete(pid, &metav1.DeleteOptions{})

		log.Debugf("Waiting for pod %v", pid)
		// Start watching for new events related to the pod we created.
		if err = v.waitForPod(pid); err != nil && err != errPodTerminated {
			return nil, err
		}
		podErr := err

		log.Debugf("Gathering logs for %v", pid)
		output, err := v.podi.GetLogs(pid, &corev1.PodLogOptions{}).Stream()
		if err != nil {
			return nil, err
		}
		defer output.Close()

		bits, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		log.Debugf("Read: %v", bits)

		if podErr == errPodTerminated {
			return nil, errors.New(string(bits))
		}
		return bytes.NewReader(bits), nil
	}
}

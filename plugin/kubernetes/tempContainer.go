package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/puppetlabs/wash/activity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// A temporary container with tools for handling cleanup.
type tempContainer struct {
	pod  *corev1.Pod
	podi typedv1.PodInterface
}

// Create a container that mounts a pvc to a default mountpoint and waits for 7 days.
func createContainer(ctx context.Context, podi typedv1.PodInterface, volumeClaim, mountpoint string) (c tempContainer, err error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "wash",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "busybox",
					Image: "busybox",
					Args:  []string{"sleep", "604800"},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1m"),
							corev1.ResourceMemory: resource.MustParse("10Mi"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeClaim,
							MountPath: mountpoint,
							ReadOnly:  true,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: volumeClaim,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: volumeClaim,
							ReadOnly:  true,
						},
					},
				},
			},
		},
	}

	c.podi = podi
	c.pod, err = podi.Create(ctx, pod, metav1.CreateOptions{})
	return
}

var errPodTerminated = errors.New("Pod terminated unexpectedly")

func (c *tempContainer) waitOnCreation(ctx context.Context) error {
	watchOpts := metav1.ListOptions{FieldSelector: "metadata.name=" + c.pod.Name}
	watcher, err := c.podi.Watch(ctx, watchOpts)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	ch := watcher.ResultChan()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return fmt.Errorf("Channel error waiting for pod %v: %v", c, e)
			}
			switch e.Type {
			case watch.Modified:
				switch e.Object.(*corev1.Pod).Status.Phase {
				case corev1.PodRunning:
					// Success, we have a running pod.
					return nil
				case corev1.PodSucceeded:
					return errPodTerminated
				case corev1.PodFailed:
					return errPodTerminated
				case corev1.PodUnknown:
					activity.Record(ctx, "Unknown state for pod %v: %v", c, e.Object)
				}
			case watch.Error:
				return fmt.Errorf("Pod %v errored: %v", c, e.Object)
			}
		case <-time.After(30 * time.Second):
			return fmt.Errorf("Timed out waiting for pod %v", c)
		}
	}
}

func (c *tempContainer) delete(ctx context.Context) error {
	var deleteImmediately int64 = 0
	return c.podi.Delete(ctx, c.pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &deleteImmediately})
}

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

type containerLogFile struct {
	plugin.EntryBase
	namespace, podName, containerName string
	client                            *k8s.Clientset
}

func newContainerLogFile(container *container) *containerLogFile {
	clf := &containerLogFile{
		EntryBase: plugin.NewEntry("log"),
	}
	clf.namespace = container.pod.Namespace
	clf.podName = container.pod.Name
	clf.containerName = container.Name()
	clf.client = container.client
	return clf
}

func (clf *containerLogFile) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(clf, "log").IsSingleton()
}

func (clf *containerLogFile) Read(ctx context.Context) ([]byte, error) {
	logOptions := corev1.PodLogOptions{
		Container: clf.containerName,
	}
	req := clf.client.CoreV1().Pods(clf.namespace).GetLogs(clf.podName, &logOptions)
	rdr, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	var n int64
	if n, err = buf.ReadFrom(rdr); err != nil {
		return nil, fmt.Errorf("unable to read logs: %v", err)
	}
	activity.Record(ctx, "Read %v bytes of %v log", n, clf.containerName)

	return buf.Bytes(), nil
}

func (clf *containerLogFile) Stream(ctx context.Context) (io.ReadCloser, error) {
	var tailLines int64 = 10
	logOptions := corev1.PodLogOptions{
		Container: clf.containerName,
		Follow:    true,
		TailLines: &tailLines,
	}
	req := clf.client.CoreV1().Pods(clf.namespace).GetLogs(clf.podName, &logOptions)
	return req.Stream(ctx)
}

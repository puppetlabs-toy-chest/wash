package kubernetes

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type pod struct {
	plugin.EntryBase
	client    *k8s.Clientset
	config    *rest.Config
	ns        string
	startTime time.Time
}

func newPod(client *k8s.Clientset, config *rest.Config, ns string, p *corev1.Pod) *pod {
	pd := &pod{
		EntryBase: plugin.NewEntry(p.Name),
		client:    client,
		config:    config,
		ns:        ns,
		startTime: p.CreationTimestamp.Time,
	}

	return pd
}

func (p *pod) Metadata(ctx context.Context) (plugin.MetadataMap, error) {
	pd, err := p.client.CoreV1().Pods(p.ns).Get(p.Name(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return plugin.ToMetadata(pd), nil
}

func (p *pod) Attr() plugin.Attributes {
	return plugin.Attributes{
		Ctime: p.startTime,
		Mtime: time.Now(),
		Atime: p.startTime,
		Size:  plugin.SizeUnknown,
	}
}

func (p *pod) Open(ctx context.Context) (plugin.SizedReader, error) {
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{})
	rdr, err := req.Stream()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	var n int64
	if n, err = buf.ReadFrom(rdr); err != nil {
		return nil, err
	}
	log.Debugf("Read %v bytes of %v log", n, p.Name())
	return bytes.NewReader(buf.Bytes()), nil
}

func (p *pod) Stream(ctx context.Context) (io.Reader, error) {
	var tailLines int64 = 10
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{Follow: true, TailLines: &tailLines})
	return req.Stream()
}

func (p *pod) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (io.Reader, error) {
	execRequest := p.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(p.Name()).
		Namespace(p.ns).
		SubResource("exec").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("command", cmd)

	for _, arg := range args {
		execRequest = execRequest.Param("command", arg)
	}

	exec, err := remotecommand.NewSPDYExecutor(p.config, "POST", execRequest.URL())
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()
	go func() {
		plugin.LogErr(exec.Stream(remotecommand.StreamOptions{
			Stdout: w,
			Stderr: w,
		}))
		plugin.LogErr(w.Close())
	}()
	return r, nil
}

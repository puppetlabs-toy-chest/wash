package kubernetes

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	k8exec "k8s.io/client-go/util/exec"
)

type pod struct {
	plugin.EntryBase
	client *k8s.Clientset
	config *rest.Config
	ns     string
}

func newPod(ctx context.Context, parent plugin.Entry, client *k8s.Clientset, config *rest.Config, ns string, p *corev1.Pod) (*pod, error) {
	pd := &pod{
		EntryBase: parent.NewEntry(p.Name),
		client:    client,
		config:    config,
		ns:        ns,
	}
	pd.DisableDefaultCaching()

	pdInfo := podInfoResult{
		pd: p,
	}
	logContent, err := pd.fetchLogContent(ctx)
	if err != nil {
		return nil, err
	}
	pdInfo.logContent = logContent

	meta := pdInfo.toMeta()
	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(p.CreationTimestamp.Time).
		SetAtime(attr.Ctime()).
		SetSize(uint64(meta["LogSize"].(int))).
		SetMeta(meta)
	pd.SetAttributes(attr)

	return pd, nil
}

type podInfoResult struct {
	pd         *corev1.Pod
	logContent []byte
}

func (pdInfo podInfoResult) toMeta() plugin.EntryMetadata {
	meta := plugin.ToMeta(pdInfo.pd)
	meta["LogSize"] = len(pdInfo.logContent)
	return meta
}

func (p *pod) fetchLogContent(ctx context.Context) ([]byte, error) {
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
	journal.Record(ctx, "Read %v bytes of %v log", n, p.Name())
	return buf.Bytes(), nil
}

func (p *pod) cachedPodInfo(ctx context.Context) (podInfoResult, error) {
	cachedPdInfo, err := plugin.CachedOp("PodInfo", p, 15*time.Second, func() (interface{}, error) {
		result := podInfoResult{}
		pd, err := p.client.CoreV1().Pods(p.ns).Get(p.Name(), metav1.GetOptions{})
		if err != nil {
			return result, err
		}
		result.pd = pd
		logContent, err := p.fetchLogContent(ctx)
		if err != nil {
			return result, err
		}
		result.logContent = logContent
		return result, nil
	})
	if err != nil {
		return podInfoResult{}, err
	}

	return cachedPdInfo.(podInfoResult), nil
}

func (p *pod) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	pdInfo, err := p.cachedPodInfo(ctx)
	if err != nil {
		return nil, err
	}

	return pdInfo.toMeta(), nil
}

func (p *pod) Open(ctx context.Context) (plugin.SizedReader, error) {
	pdInfo, err := p.cachedPodInfo(ctx)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(pdInfo.logContent), nil
}

func (p *pod) Stream(ctx context.Context) (io.Reader, error) {
	var tailLines int64 = 10
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{Follow: true, TailLines: &tailLines})
	return req.Stream()
}

func (p *pod) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecResult, error) {
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

	if opts.Stdin != nil {
		execRequest = execRequest.Param("stdin", "true")
	}

	execResult := plugin.ExecResult{}

	executor, err := remotecommand.NewSPDYExecutor(p.config, "POST", execRequest.URL())
	if err != nil {
		return execResult, errors.Wrap(err, "kubernetes.pod.Exec request")
	}

	outputCh, stdout, stderr := plugin.CreateExecOutputStreams(ctx)
	exitcode := 0
	go func() {
		streamOpts := remotecommand.StreamOptions{Stdout: stdout, Stderr: stderr, Stdin: opts.Stdin}
		err = executor.Stream(streamOpts)
		journal.Record(ctx, "Exec on %v complete: %v", p.Name(), err)
		if exerr, ok := err.(k8exec.ExitError); ok {
			exitcode = exerr.ExitStatus()
			err = nil
		}

		stdout.CloseWithError(err)
		stderr.CloseWithError(err)
	}()

	execResult.OutputCh = outputCh
	execResult.ExitCodeCB = func() (int, error) {
		return exitcode, nil
	}

	return execResult, nil
}

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/puppetlabs/wash/activity"
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

func newPod(ctx context.Context, client *k8s.Clientset, config *rest.Config, ns string, p *corev1.Pod) (*pod, error) {
	pd := &pod{
		EntryBase: plugin.NewEntry(p.Name),
		client:    client,
		config:    config,
		ns:        ns,
	}
	pd.DisableDefaultCaching()

	pdInfo := podInfoResult{
		pd:         p,
		logContent: pd.fetchLogContent(ctx),
	}

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

func (p *pod) fetchLogContent(ctx context.Context) []byte {
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{})
	rdr, err := req.Stream()
	if err != nil {
		return []byte(fmt.Sprintf("unable to access logs: %v", err))
	}
	var buf bytes.Buffer
	var n int64
	if n, err = buf.ReadFrom(rdr); err != nil {
		return []byte(fmt.Sprintf("unable to read logs: %v", err))
	}
	activity.Record(ctx, "Read %v bytes of %v log", n, p.Name())
	return buf.Bytes()
}

func (p *pod) cachedPodInfo(ctx context.Context) (podInfoResult, error) {
	cachedPdInfo, err := plugin.CachedOp(ctx, "PodInfo", p, 15*time.Second, func() (interface{}, error) {
		result := podInfoResult{}
		pd, err := p.client.CoreV1().Pods(p.ns).Get(p.Name(), metav1.GetOptions{})
		if err != nil {
			return result, err
		}
		result.pd = pd
		result.logContent = p.fetchLogContent(ctx)
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

func (p *pod) Stream(ctx context.Context) (io.ReadCloser, error) {
	var tailLines int64 = 10
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{Follow: true, TailLines: &tailLines})
	return req.Stream()
}

func (p *pod) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (*plugin.RunningCommand, error) {
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

	executor, err := remotecommand.NewSPDYExecutor(p.config, "POST", execRequest.URL())
	if err != nil {
		return nil, errors.Wrap(err, "kubernetes.pod.Exec request")
	}

	cmdObj := plugin.NewRunningCommand(ctx)

	// If using a Tty, create an input stream that allows us to send Ctrl-C to end execution;
	// when a Tty is allocated commands expect user input and will respond to control signals.
	stdin := opts.Stdin
	if opts.Tty {
		r, w := io.Pipe()
		if stdin != nil {
			stdin = io.MultiReader(stdin, r)
		} else {
			stdin = r
		}

		cmdObj.SetStopFunc(func() {
			// Close the response on context cancellation. Copying will block until there's more to
			// read from the exec output. For an action with no more output it may never return.
			// Append Ctrl-C to input to signal end of execution.
			_, err := w.Write([]byte{0x03})
			activity.Record(ctx, "Sent ETX on context termination: %v", err)
			w.Close()
		})
	}

	go func() {
		streamOpts := remotecommand.StreamOptions{
			Stdout: cmdObj.Stdout(),
			Stderr: cmdObj.Stderr(),
			Stdin: stdin,
			Tty: opts.Tty,
		}
		err = executor.Stream(streamOpts)
		activity.Record(ctx, "Exec on %v complete: %v", p.Name(), err)
		if exerr, ok := err.(k8exec.ExitError); ok {
			cmdObj.SetExitCode(exerr.ExitStatus())
			err = nil
		}

		cmdObj.CloseStreams(err)
	}()

	return cmdObj, nil
}

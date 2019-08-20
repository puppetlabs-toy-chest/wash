package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
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
	}
	pd.client = client
	pd.config = config
	pd.ns = ns

	pd.
		Attributes().
		SetCrtime(p.CreationTimestamp.Time).
		SetAtime(p.CreationTimestamp.Time).
		SetMeta(plugin.ToJSONObject(p))

	return pd, nil
}

func (p *pod) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(p, "pod").
		SetMetaAttributeSchema(corev1.Pod{})
}

func (p *pod) Open(ctx context.Context) (plugin.SizedReader, error) {
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{})
	rdr, err := req.Stream()
	if err != nil {
		return nil, fmt.Errorf("unable to access logs: %v", err)
	}
	var buf bytes.Buffer
	var n int64
	if n, err = buf.ReadFrom(rdr); err != nil {
		return nil, fmt.Errorf("unable to read logs: %v", err)
	}
	activity.Record(ctx, "Read %v bytes of %v log", n, p.Name())
	return bytes.NewReader(buf.Bytes()), nil
}

func (p *pod) Stream(ctx context.Context) (io.ReadCloser, error) {
	var tailLines int64 = 10
	req := p.client.CoreV1().Pods(p.ns).GetLogs(p.Name(), &corev1.PodLogOptions{Follow: true, TailLines: &tailLines})
	return req.Stream()
}

func (p *pod) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
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

	execCmd := plugin.NewExecCommand(ctx)

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

		execCmd.SetStopFunc(func() {
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
			Stdout: execCmd.Stdout(),
			Stderr: execCmd.Stderr(),
			Stdin:  stdin,
			Tty:    opts.Tty,
		}
		err = executor.Stream(streamOpts)
		activity.Record(ctx, "Exec on %v complete: %v", p.Name(), err)
		if err == nil {
			execCmd.SetExitCode(0)
		} else if exerr, ok := err.(k8exec.ExitError); ok {
			execCmd.SetExitCode(exerr.ExitStatus())
			err = nil
		} else {
			// Set the exit code error so that callers don't block
			// when trying to retrieve the command's exit code
			execCmd.SetExitCodeErr(err)
		}
		execCmd.CloseStreamsWithError(err)
	}()

	return execCmd, nil
}

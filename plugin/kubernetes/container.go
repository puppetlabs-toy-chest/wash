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

type container struct {
	plugin.EntryBase
	client *k8s.Clientset
	config *rest.Config
	ns     string
	pod    *corev1.Pod
}

func newContainer(ctx context.Context, client *k8s.Clientset, config *rest.Config, ns string, c *corev1.Container, p *corev1.Pod) (*container, error) {
	cntnr := &container{
		EntryBase: plugin.NewEntry(c.Name),
	}
	cntnr.client = client
	cntnr.config = config
	cntnr.ns = ns
	cntnr.pod = p

	// Find when the container was started; set this as the creation time
	for _, ecs := range cntnr.pod.Status.ContainerStatuses {
		if ecs.Name == cntnr.EntryBase.Name() && ecs.State.Running != nil {
			creationTimestamp := ecs.State.Running.StartedAt.Time
			cntnr.
				Attributes().
				SetAtime(creationTimestamp).
				SetCrtime(creationTimestamp).
				SetMtime(creationTimestamp)
			break
		}
	}

	cntnr.
		Attributes().
		SetMeta(plugin.ToJSONObject(c))

	return cntnr, nil
}

func (c *container) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(c, "container").
		SetMetaAttributeSchema(corev1.Container{})
}

func (c *container) Read(ctx context.Context) ([]byte, error) {
	logOptions := corev1.PodLogOptions{
		Container: c.Name(),
	}
	req := c.client.CoreV1().Pods(c.ns).GetLogs(c.pod.Name, &logOptions)
	rdr, err := req.Stream()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	var n int64
	if n, err = buf.ReadFrom(rdr); err != nil {
		return nil, fmt.Errorf("unable to read logs: %v", err)
	}
	activity.Record(ctx, "Read %v bytes of %v log", n, c.Name())

	return buf.Bytes(), nil
}

func (c *container) Stream(ctx context.Context) (io.ReadCloser, error) {
	var tailLines int64 = 10
	logOptions := corev1.PodLogOptions{
		Container: c.Name(),
		Follow:    true,
		TailLines: &tailLines,
	}
	req := c.client.CoreV1().Pods(c.ns).GetLogs(c.pod.Name, &logOptions)
	return req.Stream()
}

func (c *container) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	execRequest := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(c.pod.Name).
		Namespace(c.ns).
		SubResource("exec").
		Param("command", cmd).
		Param("container", c.Name()).
		Param("stderr", "true").
		Param("stdout", "true")

	for _, arg := range args {
		execRequest = execRequest.Param("command", arg)
	}

	if opts.Stdin != nil {
		execRequest = execRequest.Param("stdin", "true")
	}

	executor, err := remotecommand.NewSPDYExecutor(c.config, "POST", execRequest.URL())
	if err != nil {
		return nil, errors.Wrap(err, "kubernetes.container.Exec request")
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
		activity.Record(ctx, "Exec on %v complete: %v", c.Name(), err)
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

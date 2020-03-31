package kubernetes

import (
	"context"
	"io"

	"github.com/puppetlabs/wash/activity"
	corev1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// A general purpose container object that implements executing commands.
// If the pod contains only a single container, the container object may be omitted.
type containerBase struct {
	client    *k8s.Clientset
	config    *rest.Config
	pod       *corev1.Pod
	container *corev1.Container
}

// Create an executor to run a command using the provided options and context. If you want
// synchronous results, call `Stream. For asynchronous results call `AsyncStream`.
func (c *containerBase) newExecutor(ctx context.Context, cmd string, args []string, opts remotecommand.StreamOptions) (executor, error) {
	execRequest := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(c.pod.Name).
		Namespace(c.pod.Namespace).
		SubResource("exec").
		Param("command", cmd)

	if c.container != nil {
		execRequest = execRequest.Param("container", c.container.Name)
	}

	if opts.Stdout != nil {
		execRequest = execRequest.Param("stdout", "true")
	}

	if opts.Stderr != nil {
		execRequest = execRequest.Param("stderr", "true")
	}

	if opts.Stdin != nil || opts.Tty {
		execRequest = execRequest.Param("stdin", "true")
	}

	if opts.Tty {
		execRequest = execRequest.Param("tty", "true")
	}

	for _, arg := range args {
		execRequest = execRequest.Param("command", arg)
	}

	e, err := remotecommand.NewSPDYExecutor(c.config, "POST", execRequest.URL())
	return executor{ctx: ctx, exec: e, opts: opts, name: c.String()}, err
}

func (c *containerBase) String() string {
	s := c.pod.Namespace + "/" + c.pod.Name
	if c.container != nil {
		s += "/" + c.container.Name
	}
	return s
}

type executor struct {
	ctx  context.Context
	exec remotecommand.Executor
	opts remotecommand.StreamOptions
	name string
}

func (e executor) Stream() error {
	return e.exec.Stream(e.opts)
}

func (e executor) AsyncStream(errHandler func(error)) (cleanup func()) {
	// Track when Stream finishes because the calling context may cancel even though the operation
	// has completed and cause us to invoke the "stop func".
	done := make(chan struct{})

	// If using a Tty, create an input stream that allows us to send Ctrl-C to end execution;
	// when a Tty is allocated commands expect user input and will respond to control signals.
	opts := e.opts
	if opts.Tty {
		r, w := io.Pipe()
		if opts.Stdin != nil {
			opts.Stdin = io.MultiReader(opts.Stdin, r)
		} else {
			opts.Stdin = r
		}

		cleanup = func() {
			// Close the response on context cancellation. Copying will block until there's more to
			// read from the exec output. For an action with no more output it may never return.
			// Append Ctrl-C to input to signal end of execution.
			select {
			case <-done:
				// Passthrough, output completed so no need to cancel.
			default:
				// Only cancel when Stream has not completed. If we cancel but Stream has completed
				// we get an error while trying to copy the Write that can surface in multiple ways.
				_, err := w.Write([]byte{0x03})
				activity.Record(e.ctx, "Sent ETX on context termination: %v", err)
			}
			w.Close()
		}
	}

	go func() {
		err := e.exec.Stream(opts)
		close(done)
		activity.Record(e.ctx, "Exec on %v complete: %v", e.name, err)
		errHandler(err)
	}()

	return
}

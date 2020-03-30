package kubernetes

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/volume"
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
		SetPartialMetadata(c)

	return cntnr, nil
}

func (c *container) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(c, "container").
		SetPartialMetadataSchema(corev1.Container{})
}

func (c *container) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&containerLogFile{}).Schema(),
		(&plugin.MetadataJSONFile{}).Schema(),
		(&volume.FS{}).Schema(),
	}
}

func (c *container) List(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: May be worth creating a helper that makes it easy to create
	// read-only files. Lots of shared code between these two.
	cm, err := plugin.NewMetadataJSONFile(ctx, c)
	if err != nil {
		return nil, err
	}
	clf := newContainerLogFile(c)

	// Include a view of the remote filesystem using volume.FS. Use a small maxdepth because
	// VMs can have lots of files and Exec is fast.
	return []plugin.Entry{clf, cm, volume.NewFS(ctx, "fs", c, 3)}, nil
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

	if opts.Stdin != nil || opts.Tty {
		execRequest = execRequest.Param("stdin", "true")
	}

	if opts.Tty {
		execRequest = execRequest.Param("tty", "true")
	}

	executor, err := remotecommand.NewSPDYExecutor(c.config, "POST", execRequest.URL())
	if err != nil {
		return nil, errors.Wrap(err, "kubernetes.container.Exec request")
	}

	execCmd := plugin.NewExecCommand(ctx)

	// Track when Stream finishes because the calling context may cancel even though the operation
	// has completed and cause us to invoke the "stop func".
	done := make(chan struct{})

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
			select {
			case <-done:
				// Passthrough, output completed so no need to cancel.
			default:
				// Only cancel when Stream has not completed. If we cancel but Stream has completed, then
				// we get an error while trying to copy the Write
				//   E0330 13:54:35.930448   49254 v2.go:105] EOF
				// from https://github.com/kubernetes/client-go/blob/v10.0.0/tools/remotecommand/v2.go#L105
				// where it logs the error. I tried to change the log destination, but
				// https://github.com/kubernetes/client-go/issues/18 seems to preclude that solution.
				// Calling `klog.SetOutput` didn't do anything.
				_, err := w.Write([]byte{0x03})
				activity.Record(ctx, "Sent ETX on context termination: %v", err)
			}
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
		close(done)
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

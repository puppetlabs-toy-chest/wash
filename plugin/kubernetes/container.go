package kubernetes

import (
	"context"

	"github.com/pkg/errors"
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
	containerBase
}

func newContainer(ctx context.Context, client *k8s.Clientset, config *rest.Config, c *corev1.Container, p *corev1.Pod) (*container, error) {
	cntnr := &container{
		EntryBase: plugin.NewEntry(c.Name),
	}
	cntnr.client = client
	cntnr.config = config
	cntnr.pod = p
	cntnr.container = c

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
	execCmd := plugin.NewExecCommand(ctx)
	executor, err := c.newExecutor(ctx, cmd, args, remotecommand.StreamOptions{
		Stdout: execCmd.Stdout(),
		Stderr: execCmd.Stderr(),
		Stdin:  opts.Stdin,
		Tty:    opts.Tty,
	})
	if err != nil {
		return nil, errors.Wrap(err, "kubernetes.container.Exec request")
	}

	errHandler := func(err error) {
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
	}

	cleanup := executor.AsyncStream(errHandler)
	execCmd.SetStopFunc(cleanup)
	return execCmd, nil
}

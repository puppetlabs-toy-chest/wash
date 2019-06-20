package plugin

import (
	"bytes"
	"context"
	"fmt"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin/internal"
)

// externalPluginScript represents an external plugin's script
type externalPluginScript interface {
	Path() string
	InvokeAndWait(ctx context.Context, method string, entry *externalPluginEntry, args ...string) ([]byte, error)
	NewInvocation(ctx context.Context, method string, entry *externalPluginEntry, args ...string) *internal.Command
}

type externalPluginScriptImpl struct {
	path string
}

func (s externalPluginScriptImpl) Path() string {
	return s.path
}

// InvokeAndWait invokes method on entry by shelling out to the plugin script.
// It waits for the script to exit, then returns its standard output.
func (s externalPluginScriptImpl) InvokeAndWait(
	ctx context.Context,
	method string,
	entry *externalPluginEntry,
	args ...string,
) ([]byte, error) {
	cmd := s.NewInvocation(ctx, method, entry, args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetStdout(&stdoutBuf)
	cmd.SetStderr(&stderrBuf)
	activity.Record(ctx, "Invoking %v", cmd)
	err := cmd.Run()
	exitCode := cmd.ProcessState().ExitCode()
	if exitCode < 0 {
		return nil, err
	}
	stderr := stderrBuf.String()
	if exitCode == 0 {
		activity.Record(ctx, "stdout: %v", stdoutBuf.String())
		if len(stderr) != 0 {
			activity.Record(ctx, "stderr: %v", stderr)
		}
	} else {
		return nil, fmt.Errorf("script returned a non-zero exit code of %v. stderr output: %v", exitCode, stderr)
	}
	return stdoutBuf.Bytes(), nil
}

func (s externalPluginScriptImpl) NewInvocation(
	ctx context.Context,
	method string,
	entry *externalPluginEntry,
	args ...string,
) *internal.Command {
	if method == "init" {
		return internal.NewCommand(ctx, s.Path(), append([]string{"init"}, args...)...)
	}
	if entry == nil {
		msg := fmt.Sprintf("s.NewInvocation called with method '%v' and entry == nil", method)
		panic(msg)
	}
	return internal.NewCommand(
		ctx,
		s.Path(),
		append([]string{method, entry.id(), entry.state}, args...)...,
	)
}

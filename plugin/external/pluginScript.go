package external

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
)

// pluginScript represents an external plugin's script
type pluginScript interface {
	Path() string
	InvokeAndWait(ctx context.Context, method string, entry *pluginEntry, args ...string) (invocation, error)
	NewInvocation(ctx context.Context, method string, entry *pluginEntry, args ...string) invocation
}

// A Command object that stores output in separate stdout and stderr buffers.
type invocation interface {
	Command
	// RunAndWait should run the command, ensuring stdout and stderr are buffered, and return
	// any errors or non-zero exit codes that result from running the command.
	RunAndWait(context.Context) error
	Stdout() *bytes.Buffer
	Stderr() *bytes.Buffer
}

type invocationImpl struct {
	Command
	stdout, stderr bytes.Buffer
}

func (inv *invocationImpl) RunAndWait(ctx context.Context) error {
	inv.SetStdout(&inv.stdout)
	inv.SetStderr(&inv.stderr)

	activity.Record(ctx, "Invoking %v", inv)
	err := inv.Run()
	exitCode := inv.ExitCode()
	if exitCode < 0 {
		return newInvokeError(err.Error(), inv)
	}

	activity.Record(ctx, "stdout: %v", inv.stdout.String())
	if inv.stderr.Len() != 0 {
		activity.Record(ctx, "stderr: %v", inv.stderr.String())
	}
	if exitCode != 0 {
		return newInvokeError(fmt.Sprintf("script returned a non-zero exit code of %v", exitCode), inv)
	}
	return nil
}

func (inv *invocationImpl) Stdout() *bytes.Buffer {
	return &inv.stdout
}

func (inv *invocationImpl) Stderr() *bytes.Buffer {
	return &inv.stderr
}

func newInvokeError(msg string, inv invocation) error {
	var builder strings.Builder
	builder.WriteString(msg)
	fmt.Fprintf(&builder, "\nCOMMAND: %s", inv)
	stdout := inv.Stdout()
	if stdout.Len() > 0 {
		fmt.Fprintf(&builder, "\nSTDOUT:\n%s", strings.Trim(stdout.String(), "\n"))
	}
	if output := strings.Trim(inv.Stderr().String(), "\n"); len(output) > 0 {
		fmt.Fprintf(&builder, "\nSTDERR:\n%s", output)
	}
	return errors.New(builder.String())
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
	entry *pluginEntry,
	args ...string,
) (invocation, error) {
	inv := s.NewInvocation(ctx, method, entry, args...)
	err := inv.RunAndWait(ctx)
	return inv, err
}

func (s externalPluginScriptImpl) NewInvocation(
	ctx context.Context,
	method string,
	entry *pluginEntry,
	args ...string,
) invocation {
	if method == "init" {
		return &invocationImpl{Command: NewCommand(ctx, s.Path(), append([]string{"init"}, args...)...)}
	}
	if entry == nil {
		msg := fmt.Sprintf("s.NewInvocation called with method '%v' and entry == nil", method)
		panic(msg)
	}
	return &invocationImpl{Command: NewCommand(
		ctx,
		s.Path(),
		append([]string{method, plugin.ID(entry), entry.state}, args...)...,
	)}
}

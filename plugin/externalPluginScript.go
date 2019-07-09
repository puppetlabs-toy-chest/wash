package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin/internal"
)

// externalPluginScript represents an external plugin's script
type externalPluginScript interface {
	Path() string
	InvokeAndWait(ctx context.Context, method string, entry *externalPluginEntry, args ...string) (invocation, error)
	NewInvocation(ctx context.Context, method string, entry *externalPluginEntry, args ...string) invocation
}

type invocation struct {
	command        *internal.Command
	stdout, stderr bytes.Buffer
}

func newInvokeError(msg string, inv invocation) error {
	var builder strings.Builder
	builder.WriteString(msg)
	fmt.Fprintf(&builder, "\nCOMMAND: %s", inv.command)
	if inv.stdout.Len() > 0 {
		fmt.Fprintf(&builder, "\nSTDOUT:\n%s", strings.Trim(inv.stdout.String(), "\n"))
	}
	if output := strings.Trim(inv.stderr.String(), "\n"); len(output) > 0 {
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
	entry *externalPluginEntry,
	args ...string,
) (invocation, error) {
	inv := s.NewInvocation(ctx, method, entry, args...)
	inv.command.SetStdout(&inv.stdout)
	inv.command.SetStderr(&inv.stderr)
	activity.Record(ctx, "Invoking %v", inv.command)
	err := inv.command.Run()
	exitCode := inv.command.ProcessState().ExitCode()
	if exitCode < 0 {
		return inv, newInvokeError(err.Error(), inv)
	}

	activity.Record(ctx, "stdout: %v", inv.stdout)
	if inv.stderr.Len() != 0 {
		activity.Record(ctx, "stderr: %v", inv.stderr)
	}
	if exitCode != 0 {
		return inv, newInvokeError(fmt.Sprintf("script returned a non-zero exit code of %v", exitCode), inv)
	}
	return inv, nil
}

func (s externalPluginScriptImpl) NewInvocation(
	ctx context.Context,
	method string,
	entry *externalPluginEntry,
	args ...string,
) invocation {
	if method == "init" {
		return invocation{command: internal.NewCommand(ctx, s.Path(), append([]string{"init"}, args...)...)}
	}
	if entry == nil {
		msg := fmt.Sprintf("s.NewInvocation called with method '%v' and entry == nil", method)
		panic(msg)
	}
	return invocation{command: internal.NewCommand(
		ctx,
		s.Path(),
		append([]string{method, entry.id(), entry.state}, args...)...,
	)}
}

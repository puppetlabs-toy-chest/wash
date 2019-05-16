package plugin

import (
	"context"
	"fmt"
)


// RunningCommand represents a running command that was invoked by Execable#exec.
// Use plugin.NewRunningCommand to create these objects.
type RunningCommand struct {
	ctx           context.Context
	outputCh      chan ExecOutputChunk
	stdout        *OutputStream
	stderr        *OutputStream
	exitCodeCh    chan int
	exitCodeErrCh chan error
}

// NewRunningCommand creates a new RunningCommand object that's tied to
// the passed-in execution context.
func NewRunningCommand(ctx context.Context) *RunningCommand {
	cmd := &RunningCommand{
		ctx: ctx,
		exitCodeCh: make(chan int, 1),
		exitCodeErrCh: make(chan error, 1),
	}

	// Create the output streams
	cmd.outputCh = make(chan ExecOutputChunk)
	closer := &multiCloser{ch: cmd.outputCh, countdown: 2}
	cmd.stdout = &OutputStream{ctx: cmd.ctx, id: Stdout, ch: cmd.outputCh, closer: closer}
	cmd.stderr = &OutputStream{ctx: cmd.ctx, id: Stderr, ch: cmd.outputCh, closer: closer}

	// Ensure that the output streams are closed when the context
	// is cancelled. This guarantees that callers won't be blocked
	// when they are streaming our command's output.
	go func() {
		<-cmd.ctx.Done()
		cmd.CloseStreamsWithError(ctx.Err())
	}()

	return cmd
}

// SetStopFunc sets the function that stops the running command. stopFunc
// is called when the execution context completes to perform necessary
// termination. Hence, it should noop for a finished command.
func (cmd *RunningCommand) SetStopFunc(stopFunc func()) {
	// Thankfully, goroutines are cheap. Otherwise, mixing this in with
	// the goroutine in NewRunningCommand heavily complicates things.
	// For example, we'd have to worry about the possibility that the
	// NewRunningCommand goroutine is invoked before the Execable#Exec
	// implementation can set a stopFunc, which can happen if the context
	// is prematurely cancelled. That can result in an orphaned process
	// in some plugin APIs, which is bad.
	if stopFunc != nil {
		go func() {
			<-cmd.ctx.Done()
			stopFunc()
		}()
	}
}

// Stdout returns the command's stdout stream. Attach this to your
// plugin API's stdout stream.
func (cmd *RunningCommand) Stdout() *OutputStream {
	return cmd.stdout
}

// Stderr returns the command's stderr stream. Attach this to your
// plugin API's stderr stream.
func (cmd *RunningCommand) Stderr() *OutputStream {
	return cmd.stderr
}

// CloseStreamsWithError closes the command's stdout/stderr streams
// with the given error.
func (cmd *RunningCommand) CloseStreamsWithError(err error) {
	cmd.Stdout().CloseWithError(err)
	cmd.Stderr().CloseWithError(err)
}

// SetExitCode sets the command's exit code.
func (cmd *RunningCommand) SetExitCode(exitCode int) {
	select {
	case <-cmd.ctx.Done():
		// Don't send anything if the context is cancelled.
	default:
		cmd.exitCodeCh <- exitCode
	}
}

// SetExitCodeErr sets the exit code error, which occurs when the
// plugin API fails to fetch the comand's exit code. You should call
// this function after closing the command's output streams.
//
// NOTE: You should only use this function if your plugin API requires
// a separate request to fetch the command's exit code. Otherwise,
// use SetExitCode. See the implementation of Container#Exec in the
// Docker plugin for an example of when and how this is used.
func (cmd *RunningCommand) SetExitCodeErr(err error) {
	select {
	case <-cmd.ctx.Done():
		// Don't send anything if the context is cancelled.
	default:
		cmd.exitCodeErrCh <- err
	}
}

// StreamOutput streams the running command's output
func (cmd *RunningCommand) StreamOutput() <-chan ExecOutputChunk {
	return cmd.outputCh
}

// ExitCode returns the command's exit code. It will block until the command's
// exit code is set, or until the execution context is cancelled. ExitCode will
// return an error if it fails to fetch the command's exit code.
func (cmd *RunningCommand) ExitCode() (int, error) {
	select {
	case <-cmd.ctx.Done():
		return 0, fmt.Errorf("failed to fetch the command's exit code: %v", cmd.ctx.Err())
	case exitCode := <-cmd.exitCodeCh:
		return exitCode, nil
	case err := <-cmd.exitCodeErrCh:
		return 0, err
	}
}
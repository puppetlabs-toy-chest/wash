package plugin

import (
	"context"
	"fmt"
)

// ExecCommandImpl implements the plugin.ExecCommand interface.
// Use plugin.NewExecCommand to create instances of these objects.
type ExecCommandImpl struct {
	ctx           context.Context
	outputCh      chan ExecOutputChunk
	stdout        *OutputStream
	stderr        *OutputStream
	exitCodeCh    chan int
	exitCodeErrCh chan error
}

// NewExecCommand creates a new ExecCommandImpl object whose
// lifetime is tied to the passed-in execution context.
func NewExecCommand(ctx context.Context) *ExecCommandImpl {
	cmd := &ExecCommandImpl{
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
// termination.
func (cmd *ExecCommandImpl) SetStopFunc(stopFunc func()) {
	// Thankfully, goroutines are cheap. Otherwise, mixing this in with
	// the goroutine in NewExecCommand heavily complicates things.
	// For example, we'd have to worry about the possibility that the
	// NewExecCommand goroutine is invoked before the Execable#Exec
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
func (cmd *ExecCommandImpl) Stdout() *OutputStream {
	return cmd.stdout
}

// Stderr returns the command's stderr stream. Attach this to your
// plugin API's stderr stream.
func (cmd *ExecCommandImpl) Stderr() *OutputStream {
	return cmd.stderr
}

// CloseStreamsWithError closes the command's stdout/stderr streams
// with the given error.
func (cmd *ExecCommandImpl) CloseStreamsWithError(err error) {
	cmd.Stdout().CloseWithError(err)
	cmd.Stderr().CloseWithError(err)
}

// SetExitCode sets the command's exit code.
func (cmd *ExecCommandImpl) SetExitCode(exitCode int) {
	select {
	case <-cmd.ctx.Done():
		// Don't send anything if the context is cancelled.
	default:
		cmd.exitCodeCh <- exitCode
	}
}

// SetExitCodeErr sets the exit code error, which occurs when the
// plugin API fails to fetch the comand's exit code.
//
// NOTE: You should only use this function if your plugin API requires
// a separate request to fetch the command's exit code. Otherwise,
// use SetExitCode. See the implementation of Container#Exec in the
// Docker plugin for an example of when and how this is used.
func (cmd *ExecCommandImpl) SetExitCodeErr(err error) {
	select {
	case <-cmd.ctx.Done():
		// Don't send anything if the context is cancelled.
	default:
		cmd.exitCodeErrCh <- err
	}
}

// OutputCh implements ExecCommand#OutputCh
func (cmd *ExecCommandImpl) OutputCh() <-chan ExecOutputChunk {
	return cmd.outputCh
}

// ExitCode implements ExecCommand#ExitCode
func (cmd *ExecCommandImpl) ExitCode() (int, error) {
	select {
	case <-cmd.ctx.Done():
		return 0, fmt.Errorf("failed to fetch the command's exit code: %v", cmd.ctx.Err())
	case exitCode := <-cmd.exitCodeCh:
		return exitCode, nil
	case err := <-cmd.exitCodeErrCh:
		return 0, err
	}
}

// sealed implements ExecCommand#sealed
func (cmd *ExecCommandImpl) sealed() {
}
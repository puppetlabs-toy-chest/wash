package plugin

import (
	"context"
)


// RunningCommand represents a running command that was invoked by Execable#exec.
// Use plugin.NewRunningCommand to create these objects.
type RunningCommand struct {
	ctx        context.Context
	exitCodeCB func() (int, error)
	outputCh   chan ExecOutputChunk
	stdout     *OutputStream
	stderr     *OutputStream
}

// NewRunningCommand creates a new RunningCommand object that's tied to
// the passed-in execution context. You should call this function
// once you've verified that your plugin API has successfully
// started executing the command.
func NewRunningCommand(ctx context.Context) *RunningCommand {
	cmd := &RunningCommand{}
	cmd.ctx = ctx
	// Create the output streams
	cmd.outputCh = make(chan ExecOutputChunk)
	closer := &multiCloser{ch: cmd.outputCh, countdown: 2}
	cmd.stdout = &OutputStream{ctx: cmd.ctx, id: Stdout, ch: cmd.outputCh, closer: closer}
	cmd.stderr = &OutputStream{ctx: cmd.ctx, id: Stderr, ch: cmd.outputCh, closer: closer}
	return cmd
}

// SetStopFunc sets the function that stops the running command. stopFunc
// is called when the execution context completes to perform necessary
// termination. Hence, it should noop for a finished command.
func (cmd *RunningCommand) SetStopFunc(stopFunc func()) {
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

// CloseStreams closes the command's stdout and stderr
// streams.
func (cmd *RunningCommand) CloseStreams() {
	cmd.CloseStreamsWithError(nil)
}

// CloseStreamsWithError closes the command's stdout and stderr
// streams with the specified error.
func (cmd *RunningCommand) CloseStreamsWithError(err error) {
	cmd.Stdout().CloseWithError(err)
	cmd.Stderr().CloseWithError(err)
}

// Wait waits for the command to finish, passing in each chunk
// of the command's output to processChunk.
func (cmd *RunningCommand) Wait(processChunk func(ExecOutputChunk)) {
	for chunk := range cmd.outputCh {
		processChunk(chunk)
	}
}

// SetExitCodeCB sets the exit code callback. This is used to fetch the
// command's exit code after execution completes. You should use this if
// your plugin API requires a separate request to fetch the command's exit
// code. Otherwise, use cmd.SetExitCode to set the exit code.
//
// See the implementation of Container#Exec in the Docker plugin for an
// example of how this is used.
func (cmd *RunningCommand) SetExitCodeCB(exitCodeCB func() (int, error)) {
	cmd.exitCodeCB = exitCodeCB
}

// SetExitCode sets the command's exit code. Use this after the command's
// finished its execution.
func (cmd *RunningCommand) SetExitCode(exitCode int) {
	cmd.exitCodeCB = func() (int, error) {
		return exitCode, nil
	}
}

// ExitCode returns the command's exit code. This should be called after
// the command's finished its execution.
func (cmd *RunningCommand) ExitCode() (int, error) {
	if cmd.exitCodeCB == nil {
		panic("cmd.ExitCode called with a nil exit code callback")
	}
	return cmd.exitCodeCB()
}

package plugin

import (
	"context"
	"fmt"

	"sync"
)


// RunningCommand represents a running command that was invoked by Execable#exec.
// Use plugin.NewRunningCommand to create these objects.
type RunningCommand struct {
	ctx           context.Context
	outputCh      chan ExecOutputChunk
	stdout        *OutputStream
	stderr        *OutputStream
	mux           sync.Mutex
	streamsClosed bool
	exitCodeCh    chan int
	exitCodeErrCh chan error
}

// NewRunningCommand creates a new RunningCommand object that's tied to
// the passed-in execution context. When the execution context is cancelled,
// the returned command's stdout/stderr streams will automatically be closed.
// This means that you do not have to worry about handling dangling output
// streams if your plugin API fails to properly start the command.
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

	// Ensure that the command's properly cleaned up when the context is
	// cancelled.
	go func() {
		<-cmd.ctx.Done()
		cmd.CloseStreams(ctx.Err())
		close(cmd.exitCodeCh)
		close(cmd.exitCodeErrCh)
	}()

	return cmd
}

// SetStopFunc sets the function that stops the running command. stopFunc
// is called when the execution context completes to perform necessary
// termination. Hence, it should noop for a finished command.
func (cmd *RunningCommand) SetStopFunc(stopFunc func()) {
	if stopFunc != nil {
		// Thankfully, goroutines are cheap. Otherwise, mixing this in with
		// the goroutine in NewRunningCommand heavily complicates things.
		// For example, we'd have to worry about the possibility that the
		// NewRunningCommand goroutine is invoked before the Execable#Exec
		// implementation can set a stopFunc, which can happen if the context
		// is prematurely cancelled. That can result in an orphaned process
		// in some plugin APIs, which is bad.
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

// CloseStreams closes the command's stdout and stderr streams with
// the specified error. This should be called with err == nil when the
// command's finished its execution, or when an IO error occurs.
//
// NOTE: CloseStreams is automatically called whenever the execution
// context is cancelled, so you do not have to worry about handling
// dangling output streams.
func (cmd *RunningCommand) CloseStreams(err error) {
	// The lock is necessary because this can be invoked by multiple threads
	// (Execable#Exec + the goroutine in NewRunningComamnd)
	cmd.mux.Lock()
	defer cmd.mux.Unlock()
	if cmd.streamsClosed {
		return
	}
	cmd.Stdout().closeWithError(err)
	cmd.Stderr().closeWithError(err)
	cmd.streamsClosed = true
}

// Wait waits for the command to finish, passing in each chunk
// of the command's output to processChunk. It returns the command's
// exit code, or an error if it failed to fetch the command's exit
// code.
func (cmd *RunningCommand) Wait(processChunk func(ExecOutputChunk)) (int, error) {
	for chunk := range cmd.outputCh {
		processChunk(chunk)
	}
	select {
	// Note that the channels are only closed when the context is cancelled.
	case exitCode, ok := <-cmd.exitCodeCh:
		if !ok {
			return 0, fmt.Errorf("failed to fetch the command's exit code: %v", cmd.ctx.Err())
		}
		return exitCode, nil
	case err, ok := <-cmd.exitCodeErrCh:
		if !ok {
			return 0, fmt.Errorf("failed to fetch the command's exit code: %v", cmd.ctx.Err())
		}
		return 0, err
	}
}

// SetExitCode sets the command's exit code. You should call this
// function after closing the command's output streams.
func (cmd *RunningCommand) SetExitCode(exitCode int) {
	select {
	case <-cmd.ctx.Done():
		// Don't send anything if the context is cancelled. Otherwise,
		// we will panic trying to send to a closed channel.
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
		// Don't send anything if the context is cancelled. Otherwise,
		// we will panic trying to send to a closed channel.
	default:
		cmd.exitCodeErrCh <- err
	}
}
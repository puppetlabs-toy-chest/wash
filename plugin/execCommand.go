package plugin

import (
	"context"
)

// This file contains the implementation of the ExecCommand type

// NewExecCommand creates a new ExecCommand object that runs for
// the duration of the specified context.
//
// TODO: Clarify the comments here a bit.
func NewExecCommand(ctx context.Context) *ExecCommand {
	cmd := &ExecCommand{}
	cmd.outputCh = make(chan ExecOutputChunk)
	closer := &multiCloser{ch: cmd.outputCh, countdown: 2}
	cmd.stdout = &OutputStream{ctx: ctx, id: Stdout, ch: cmd.outputCh, closer: closer}
	cmd.stderr = &OutputStream{ctx: ctx, id: Stderr, ch: cmd.outputCh, closer: closer}
	return cmd
}

// Stdout returns the command's stdout stream
func (cmd *ExecCommand) Stdout() *OutputStream {
	return cmd.stdout
}

// Stderr returns the command's stderr stream
func (cmd *ExecCommand) Stderr() *OutputStream {
	return cmd.stderr
}

// CloseStreams closes the command's stdout and stderr
// streams.
func (cmd *ExecCommand) CloseStreams() {
	cmd.CloseStreamsWithError(nil)
}

// CloseStreamsWithError closes the command's stdout and stderr
// streams with the specified error.
func (cmd *ExecCommand) CloseStreamsWithError(err error) {
	cmd.Stdout().CloseWithError(err)
	cmd.Stderr().CloseWithError(err)
}

// Wait waits for the command to finish, passing in each chunk
// of the command's output to processChunk.
func (cmd *ExecCommand) Wait(processChunk func(ExecOutputChunk)) {
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
func (cmd *ExecCommand) SetExitCodeCB(exitCodeCB func() (int, error)) {
	cmd.exitCodeCB = exitCodeCB
}

// SetExitCode sets the command's exit code. Use this after the command's
// finished its execution.
func (cmd *ExecCommand) SetExitCode(exitCode int) {
	cmd.exitCodeCB = func() (int, error) {
		return exitCode, nil
	}
}

// ExitCode returns the command's exit code. This should be called after
// the command's finished its execution.
func (cmd *ExecCommand) ExitCode() (int, error) {
	if cmd.exitCodeCB == nil {
		panic("cmd.ExitCode called with a nil exit code callback")
	}
	return cmd.exitCodeCB()
}

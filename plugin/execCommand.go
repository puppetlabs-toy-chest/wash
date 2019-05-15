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
	// TODO: Stop the command here upon context cancellation
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
// streams. Use this to signal that execution is
// complete.
func (cmd *ExecCommand) CloseStreams() {
	cmd.CloseStreamsWithError(nil)
}

// CloseStreamsWithError closes the command's stdout and stderr
// streams with the specified error. Use this to signal that
// execution is complete.
func (cmd *ExecCommand) CloseStreamsWithError(err error) {
	cmd.stdout.CloseWithError(err)
	cmd.stderr.CloseWithError(err)
}

// Wait waits for the command to finish, passing in each chunk
// of the command's output to processChunk.
func (cmd *ExecCommand) Wait(processChunk func(ExecOutputChunk)) {
	for chunk := range cmd.outputCh {
		processChunk(chunk)
	}
}

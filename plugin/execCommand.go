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
	cmd.OutputCh = make(chan ExecOutputChunk)
	closer := &multiCloser{ch: cmd.OutputCh, countdown: 2}
	cmd.stdout = &OutputStream{ctx: ctx, id: Stdout, ch: cmd.OutputCh, closer: closer}
	cmd.stderr = &OutputStream{ctx: ctx, id: Stderr, ch: cmd.OutputCh, closer: closer}
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
// streams. It is used to signal that execution is
// complete.
func (cmd *ExecCommand) CloseStreams() {
	cmd.CloseStreamsWithError(nil)
}

// CloseStreamsWithError closes the command's stdout and stderr
// streams with the specified error. It is used to signal that
// execution is complete.
func (cmd *ExecCommand) CloseStreamsWithError(err error) {
	cmd.stdout.CloseWithError(err)
	cmd.stderr.CloseWithError(err)
}

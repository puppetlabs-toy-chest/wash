package plugin

import (
	"context"
	"sync"
	"time"
)

const (
	// StdoutID represents Stdout
	StdoutID = iota
	// StderrID represents Stderr
	StderrID
)

// OutputStream represents stdout/stderr.
type OutputStream struct {
	ctx        context.Context
	sentCtxErr bool
	id         int8
	ch         chan ExecOutputChunk
	closer     *multiCloser
}

func (s *OutputStream) sendData(data string) {
	s.ch <- ExecOutputChunk{StreamID: s.id, Timestamp: time.Now(), Data: data}
}

func (s *OutputStream) sendError(err error) {
	s.ch <- ExecOutputChunk{StreamID: s.id, Timestamp: time.Now(), Err: err}
}

func (s *OutputStream) Write(data []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		s.sendError(s.ctx.Err())
		s.sentCtxErr = true
		return 0, s.ctx.Err()
	default:
		s.sendData(string(data))
		return len(data), nil
	}
}

// Close ensures the channel is closed when the last OutputStream is closed.
func (s *OutputStream) Close() {
	s.closer.Close()
}

// CloseWithError sends the given error before calling Close()
func (s *OutputStream) CloseWithError(err error) {
	if err != nil {
		// Avoid re-sending ctx.Err() if it was already sent
		// by OutputStream#Write
		if err != s.ctx.Err() || !s.sentCtxErr {
			s.sendError(err)
		}
	}

	s.Close()
}

type multiCloser struct {
	mux       sync.Mutex
	ch        chan ExecOutputChunk
	countdown int
}

func (c *multiCloser) Close() {
	c.mux.Lock()
	c.countdown--
	if c.countdown == 0 {
		close(c.ch)
	}
	c.mux.Unlock()
}

// CreateExecOutputStreams creates a pair of writers representing stdout
// and stderr. They are used to transfer chunks of the Exec'ed cmd's
// output in the order they're received by the corresponding API. The
// writers maintain the ordering by writing to a channel.
//
// This method returns outputCh, stdout, and stderr, respectively.
func CreateExecOutputStreams(ctx context.Context) (<-chan ExecOutputChunk, *OutputStream, *OutputStream) {
	outputCh := make(chan ExecOutputChunk)
	closer := &multiCloser{ch: outputCh, countdown: 2}

	stdout := &OutputStream{ctx: ctx, id: StdoutID, ch: outputCh, closer: closer}
	stderr := &OutputStream{ctx: ctx, id: StderrID, ch: outputCh, closer: closer}

	return outputCh, stdout, stderr
}

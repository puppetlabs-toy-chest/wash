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

func (s *OutputStream) sendData(timestamp time.Time, data string) {
	s.ch <- ExecOutputChunk{StreamID: s.id, Timestamp: timestamp, Data: data}
}

func (s *OutputStream) sendError(timestamp time.Time, err error) {
	s.ch <- ExecOutputChunk{StreamID: s.id, Timestamp: timestamp, Err: err}
}

// WriteWithTimestamp writes the given data with the specified timestamp
func (s *OutputStream) WriteWithTimestamp(timestamp time.Time, data []byte) error {
	select {
	case <-s.ctx.Done():
		s.sendError(timestamp, s.ctx.Err())
		s.sentCtxErr = true
		return s.ctx.Err()
	default:
		s.sendData(timestamp, string(data))
		return nil
	}
}

func (s *OutputStream) Write(data []byte) (int, error) {
	err := s.WriteWithTimestamp(time.Now(), data)
	return len(data), err
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
			s.sendError(time.Now(), err)
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

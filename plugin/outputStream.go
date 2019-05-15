package plugin

import (
	"context"
	"sync"
	"time"
)

// OutputStream represents stdout/stderr.
type OutputStream struct {
	ctx        context.Context
	sentCtxErr bool
	id         ExecPacketType
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
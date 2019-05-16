package plugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// NOTE: Some of the assertion helpers in this file take in a suite.Suite
// object as a parameter because they are also used by the ExecCommandImpl
// tests.

type OutputStreamTestSuite struct {
	suite.Suite
}

func newOutputStream(ctx context.Context) *OutputStream {
	ch := make(chan ExecOutputChunk, 1)
	return &OutputStream{
		ctx: ctx,
		id: Stdout,
		ch: ch,
		closer: &multiCloser{
			ch: ch,
			countdown: 1,
		},
	}
}

func EqualChunk(suite suite.Suite, expected ExecOutputChunk, actual ExecOutputChunk) bool {
	expectedStreamName := expected.StreamID
	eqlStreamName := suite.Equal(
		expectedStreamName,
		actual.StreamID,
		fmt.Sprintf("The sent ExecOutputChunk should have come from the %v stream", expectedStreamName),
	)

	suite.NotZero(actual.Timestamp, "The sent ExecOutputChunk should contain a timestamp")

	var eqlPacket bool
	if expected.Data != "" {
		eqlPacket = suite.Equal(
			expected.Data,
			actual.Data,
			"The sent ExecOutputChunk should contain the expected data",
		)
	} else {
		eqlPacket = suite.Equal(
			expected.Err,
			actual.Err,
			"The sent ExecOutputChunk shoudld contain the expected error",
		)
	}

	return eqlStreamName && eqlPacket
}

// We choose not to add tests for WriteWithTimestamp because doing so
// would complicate the test suite for little gain. Instead, since Write
// calls WriteWithTimestamp, it is reasonable to assume that if the tests
// for Write pass then so, too, do the tests for WriteWithTimestamp.
func (suite *OutputStreamTestSuite) TestWrite() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	stream := newOutputStream(ctx)
	
	// Test a successful write
	data := []byte("data")
	nw, writeErr := stream.Write(data)
	if suite.NoError(writeErr) {
		suite.Equal(len(data), nw, "Write should return the number of written bytes")
	}
	select {
	case chunk := <-stream.ch:
		EqualChunk(
			suite.Suite,
			ExecOutputChunk{StreamID: Stdout, Data: string(data)},
			chunk,
		)
	default:
		suite.Fail("Write did not write any data")
	}

	// Test that the write errors when the context is cancelled.
	cancelFunc()
	_, writeErr = stream.Write(data)
	suite.EqualError(writeErr, ctx.Err().Error(), "Write should have returned the context's error")
	select {
	case chunk := <-stream.ch:
		EqualChunk(
			suite.Suite,
			ExecOutputChunk{StreamID: Stdout, Err: ctx.Err()},
			chunk,
		)
		suite.Equal(stream.sentCtxErr, true, "The stream should mark that the context's error was sent")
	default:
		suite.Fail("Write did not send the context's error")
	}
}

func assertClosedChannel(suite suite.Suite, ch <-chan ExecOutputChunk) {
	timer := time.NewTimer(1 * time.Second)
	select {
	case <-timer.C:
		suite.Fail("Timed out while waiting for the output channel to be closed")
	case chunk, ok := <-ch:
		if ok {
			suite.Fail(
				fmt.Sprintf("Expected channel to be closed; received %v instead.", chunk),
			)
		}
	}
}

func (suite *OutputStreamTestSuite) assertClosed(stream *OutputStream) {
	assertClosedChannel(suite.Suite, stream.ch)
	suite.True(stream.closed)
}

func assertSentError(suite suite.Suite, stream *OutputStream, err error) {
	timer := time.NewTimer(1 * time.Second)
	sentErrorMsg := fmt.Sprintf("Expected the error '%v' to be sent", err)
	select {
	case <-timer.C:
		suite.Fail(sentErrorMsg + ", but timed out while waiting for it")
	case chunk, ok := <-stream.ch:
		if !ok {
			suite.Fail(sentErrorMsg + ", but the channel was closed")
		} else {
			EqualChunk(
				suite,
				ExecOutputChunk{StreamID: stream.id, Err: err},
				chunk,
			)
		}
	}
}

func (suite *OutputStreamTestSuite) TestCloseWithError_NoError() {
	stream := newOutputStream(context.Background())
	stream.CloseWithError(nil)
	// Note that if an error was sent, then assertClosed would fail
	// because it'd read-in the sent error.
	suite.assertClosed(stream)
}

func (suite *OutputStreamTestSuite) TestCloseWithError_WithError() {
	stream := newOutputStream(context.Background())
	err := fmt.Errorf("an arbitrary error")
	stream.CloseWithError(err)
	assertSentError(suite.Suite, stream, err)
	suite.assertClosed(stream)
}

func (suite *OutputStreamTestSuite) TestCloseWithError_ContextError_NotSent() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	stream := newOutputStream(ctx)
	stream.CloseWithError(ctx.Err())
	assertSentError(suite.Suite, stream, ctx.Err())
	suite.assertClosed(stream)
}

func (suite *OutputStreamTestSuite) TestCloseWithError_ContextError_Sent() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	stream := newOutputStream(ctx)
	stream.sentCtxErr = true
	stream.CloseWithError(ctx.Err())
	// Note that if CloseWithError sent the passed-in ctx.Err(),
	// then assertClosed would fail because it'd read-in the sent
	// ctx.Err()
	suite.assertClosed(stream)
}

func (suite *OutputStreamTestSuite) TestCloseWithError_ConsecutiveCloses() {
	// Main thing to test here is to ensure that multiple close calls
	// won't panic by trying to close stream.ch when it is already
	// closed
	stream := newOutputStream(context.Background())
	stream.CloseWithError(nil)
	suite.assertClosed(stream)
	stream.CloseWithError(nil)
	suite.assertClosed(stream)
}

func TestOutputStream(t *testing.T) {
	suite.Run(t, new(OutputStreamTestSuite))
}

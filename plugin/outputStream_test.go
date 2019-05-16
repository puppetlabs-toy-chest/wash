package plugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

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

func (suite *OutputStreamTestSuite) EqualChunk(expected ExecOutputChunk, actual ExecOutputChunk) bool {
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
	defer func() {
		// Ensure that the context and its associated resources
		// are cleaned up
		select {
		case <-ctx.Done():
			// Pass-thru, context was already cancelled
		default:
			cancelFunc()
		}
	}()

	stream := newOutputStream(ctx)
	
	// Test a successful write
	data := []byte("data")
	nw, writeErr := stream.Write(data)
	if suite.NoError(writeErr) {
		suite.Equal(len(data), nw, "Write should return the number of written bytes")
	}
	select {
	case chunk := <-stream.ch:
		suite.EqualChunk(
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
		suite.EqualChunk(
			ExecOutputChunk{StreamID: Stdout, Err: ctx.Err()},
			chunk,
		)
		suite.Equal(stream.sentCtxErr, true, "The stream should mark that the context's error was sent")
	default:
		suite.Fail("Write did not send the context's error")
	}
}

func (suite *OutputStreamTestSuite) assertClosedChannel(ch <-chan ExecOutputChunk) {
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
	suite.assertClosedChannel(stream.ch)
	suite.True(stream.closed)
}

func (suite *OutputStreamTestSuite) assertSentError(stream *OutputStream, err error) {
	sentErrorMsg := fmt.Sprintf("Expected the error '%v' to be sent", err)
	select {
	case chunk, ok := <-stream.ch:
		if !ok {
			suite.Fail(sentErrorMsg + ", but the channel was closed")
		} else {
			suite.EqualChunk(
				ExecOutputChunk{StreamID: stream.id, Err: err},
				chunk,
			)
		}
	default:
		suite.Fail(sentErrorMsg + ", but nothing was sent on the channel")
	}
}

func (suite *OutputStreamTestSuite) TestCloseWithError_NoError() {
	stream := newOutputStream(context.Background())
	stream.CloseWithError(nil)
	suite.assertClosed(stream)
}

func (suite *OutputStreamTestSuite) TestCloseWithError_WithError() {
	stream := newOutputStream(context.Background())
	err := fmt.Errorf("an arbitrary error")
	stream.CloseWithError(err)
	suite.assertSentError(stream, err)
	suite.assertClosed(stream)
}

func (suite *OutputStreamTestSuite) TestCloseWithError_ContextError_NotSent() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	stream := newOutputStream(ctx)
	stream.CloseWithError(ctx.Err())
	suite.assertSentError(stream, ctx.Err())
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

/*
TODO: Move this test over to runningCommand.go
func (suite *OutputStreamTestSuite) TestCreateExecOutputStreams() {
	outputCh, stdout, stderr := CreateExecOutputStreams(context.Background())

	// Our simulated command alternates writing to stdout + stderr
	expectedChunksCh := make(chan ExecOutputChunk, 1)
	go func() {
		// stdout + stderr will be closed before expectedChunksCh,
		// which means that outputCh should be closed before
		// expectedChunksCh.
		defer close(expectedChunksCh)
		defer stdout.close()
		defer stderr.close()

		writeTo := func(streamName string, stream *OutputStream, data string) {
			expectedChunksCh <- ExecOutputChunk{StreamID: stream.id, Data: data}
			_, err := stream.Write([]byte(data))
			if !suite.NoError(err) {
				suite.FailNow(
					fmt.Sprintf("Unexpected error writing to %v: %v", streamName, err),
				)
			}
		}
		for i := 0; i < 5; i++ {
			data := strconv.Itoa(i)
			writeTo("stdout", stdout, data)
			writeTo("stderr", stderr, data)
		}
	}()

	for expectedChunk := range expectedChunksCh {
		timer := time.NewTimer(1 * time.Second)
		select {
		case <-timer.C:
			suite.FailNow(
				fmt.Sprintf("Timed out while waiting for chunk %v to be sent to the output channel", expectedChunk),
			)
		case chunk, ok := <-outputCh:
			if !ok {
				suite.FailNow(
					fmt.Sprintf("Expected chunk %v, but output channel was prematurely closed", expectedChunk),
				)
			}

			suite.EqualChunk(expectedChunk, chunk)
		}
	}

	// outputCh should be closed, so assert that it is
	suite.assertClosedChannel(outputCh)
}
*/

func TestOutputStream(t *testing.T) {
	suite.Run(t, new(OutputStreamTestSuite))
}

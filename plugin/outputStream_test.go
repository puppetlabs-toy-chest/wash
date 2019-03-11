package plugin

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type OutputStreamTestSuite struct {
	suite.Suite
}

func (suite *OutputStreamTestSuite) EqualChunk(expected ExecOutputChunk, actual ExecOutputChunk) bool {
	streamName := func(id int8) (string, error) {
		if id == StdoutID {
			return "stdout", nil
		}

		if id == StderrID {
			return "stderr", nil
		}

		return "", fmt.Errorf("unknown ID: %v", id)
	}

	expectedStreamName, err := streamName(expected.StreamID)
	if err != nil {
		suite.Fail("could not get the stream name of the expected chunk: %v", err)
	}

	actualStreamName, err := streamName(actual.StreamID)
	if err != nil {
		suite.Fail("actual chunk has an invalid stream ID: %v", err)
	}

	eqlStreamName := suite.Equal(
		expectedStreamName,
		actualStreamName,
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

	ch := make(chan ExecOutputChunk, 1)
	defer close(ch)

	stream := OutputStream{ctx: ctx, id: StdoutID, ch: ch}

	// Test a successful write
	data := []byte("data")
	nw, writeErr := stream.Write(data)
	if suite.NoError(writeErr) {
		suite.Equal(len(data), nw, "Write should return the number of written bytes")
	}
	select {
	case chunk := <-stream.ch:
		suite.EqualChunk(
			ExecOutputChunk{StreamID: StdoutID, Data: string(data)},
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
			ExecOutputChunk{StreamID: StdoutID, Err: ctx.Err()},
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

func (suite *OutputStreamTestSuite) TestClose() {
	ch := make(chan ExecOutputChunk)
	stream := OutputStream{ch: ch, closer: &multiCloser{ch: ch, countdown: 1}}

	stream.Close()
	suite.assertClosedChannel(stream.ch)
}

func (suite *OutputStreamTestSuite) TestCloseWithError() {
	newOutputStream := func(ctx context.Context) OutputStream {
		ch := make(chan ExecOutputChunk, 1)
		return OutputStream{ctx: ctx, id: StdoutID, ch: ch, closer: &multiCloser{ch: ch, countdown: 1}}
	}

	// Test that if err == nil, then nothing was sent to the channel
	stream := newOutputStream(context.Background())
	stream.CloseWithError(nil)
	suite.assertClosedChannel(stream.ch)

	// Useful assertion for the subsequent tests
	assertSentError := func(stream OutputStream, err error) {
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

	// Test that if err != nil, then the error is sent to the channel
	stream = newOutputStream(context.Background())
	sentErr := fmt.Errorf("an arbitrary error")
	stream.CloseWithError(sentErr)
	assertSentError(stream, sentErr)
	suite.assertClosedChannel(stream.ch)

	// Test that if err == ctx.Err(), then ctx.Err() is sent if a previous
	// Write did not send it
	ctx, cancelFunc := context.WithCancel(context.Background())
	stream = newOutputStream(ctx)
	cancelFunc()
	stream.CloseWithError(ctx.Err())
	assertSentError(stream, ctx.Err())
	suite.assertClosedChannel(stream.ch)

	// Now, test that if err == ctx.Err(), then ctx.Err() is _not_ sent if
	// a previous Write sent it
	stream = newOutputStream(ctx)
	stream.sentCtxErr = true
	stream.CloseWithError(ctx.Err())
	suite.assertClosedChannel(stream.ch)

}

func (suite *OutputStreamTestSuite) TestCreateExecOutputStreams() {
	outputCh, stdout, stderr := CreateExecOutputStreams(context.Background())

	assertStreamID := func(streamName string, expectedID int8, stream *OutputStream) {
		if !suite.Equal(expectedID, stream.id) {
			msg := fmt.Sprintf("expected %v stream to have stream ID %v, got %v instead", streamName, expectedID, stream.id)
			suite.FailNow(msg)
		}
	}
	assertStreamID("stdout", StdoutID, stdout)
	assertStreamID("stderr", StderrID, stderr)

	// Our simulated command alternates writing to stdout + stderr
	expectedChunksCh := make(chan ExecOutputChunk, 1)
	go func() {
		// stdout + stderr will be closed before expectedChunksCh,
		// which means that outputCh should be closed before
		// expectedChunksCh.
		defer close(expectedChunksCh)
		defer stdout.Close()
		defer stderr.Close()

		writeTo := func(stream *OutputStream, data string) {
			expectedChunksCh <- ExecOutputChunk{StreamID: stream.id, Data: data}
			stream.Write([]byte(data))
		}
		for i := 0; i < 5; i++ {
			data := strconv.Itoa(i)
			writeTo(stdout, data)
			writeTo(stderr, data)
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

func (suite *OutputStreamTestSuite) SetupTest() {
	// Turn off the cache in case another set of tests initialized it
	cache = nil
}

func TestOutputStream(t *testing.T) {
	suite.Run(t, new(OutputStreamTestSuite))
}

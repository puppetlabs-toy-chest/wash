package plugin

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ExecCommandImplTestSuite struct {
	suite.Suite
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

func (suite *ExecCommandImplTestSuite) SetupTest() {
	suite.ctx, suite.cancelFunc = context.WithCancel(context.Background())
}

func (suite *ExecCommandImplTestSuite) TearDownTest() {
	// Ensure that the context is cancelled to avoid dangling
	// goroutines
	suite.cancelFunc()
}

func (suite *ExecCommandImplTestSuite) NewExecCommand() *ExecCommandImpl {
	return NewExecCommand(suite.ctx)
}

func (suite *ExecCommandImplTestSuite) TestNewExecCommand_CreatesOutputStreams() {
	execCmd := suite.NewExecCommand()

	// Our simulated command alternates writing to stdout + stderr
	expectedChunksCh := make(chan ExecOutputChunk, 1)
	go func() {
		// stdout + stderr will be closed before expectedChunksCh,
		// which means that outputCh should be closed before
		// expectedChunksCh.
		defer close(expectedChunksCh)
		defer execCmd.CloseStreamsWithError(nil)

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
			writeTo("stdout", execCmd.Stdout(), data)
			writeTo("stderr", execCmd.Stderr(), data)
		}
	}()

	for expectedChunk := range expectedChunksCh {
		timer := time.NewTimer(1 * time.Second)
		select {
		case <-timer.C:
			suite.FailNow(
				fmt.Sprintf("Timed out while waiting for chunk %v to be sent to the output channel", expectedChunk),
			)
		case chunk, ok := <-execCmd.OutputCh():
			if !ok {
				suite.FailNow(
					fmt.Sprintf("Expected chunk %v, but output channel was prematurely closed", expectedChunk),
				)
			}
			EqualChunk(suite.Suite, expectedChunk, chunk)
		}
	}

	assertClosedChannel(suite.Suite, execCmd.OutputCh())
}

func (suite *ExecCommandImplTestSuite) TestNewExecCommand_CancelledContext_ClosesOutputCh() {
	execCmd := suite.NewExecCommand()
	suite.cancelFunc()
	assertSentError(suite.Suite, execCmd.Stdout(), suite.ctx.Err())
	assertSentError(suite.Suite, execCmd.Stderr(), suite.ctx.Err())
	assertClosedChannel(suite.Suite, execCmd.OutputCh())
}

func (suite *ExecCommandImplTestSuite) TestSetStopFunc_CancelledContext_StopsCommand() {
	execCmd := suite.NewExecCommand()
	stoppedCh := make(chan bool, 1)
	time.AfterFunc(1 * time.Second, func() {
		close(stoppedCh)
	})
	execCmd.SetStopFunc(func() {
		stoppedCh <- true
	})
	suite.cancelFunc()

	stopped, ok := <-stoppedCh
	if !ok {
		suite.Fail("The command was not stopped")
	} else {
		suite.True(stopped)
	}
}

func (suite *ExecCommandImplTestSuite) TestSetExitCode_CancelledContext_DoesNotSetExitCode() {
	execCmd := suite.NewExecCommand()
	suite.cancelFunc()
	execCmd.SetExitCode(1)
	select {
	case <-execCmd.exitCodeCh:
		suite.Fail("SetExitCode set the exit code on a cancelled context")
	default:
		// Exit code was not sent, so the test passed
	}
}

func (suite *ExecCommandImplTestSuite) TestSetExitCode_SetsExitCode() {
	execCmd := suite.NewExecCommand()
	execCmd.SetExitCode(1)
	select {
	case exitCode := <-execCmd.exitCodeCh:
		suite.Equal(1, exitCode)
	default:
		suite.Fail("SetExitCode did not send the exit code on exitCodeCh")
	}
}

func (suite *ExecCommandImplTestSuite) TestSetExitCodeErr_CancelledContext_DoesNotSetExitCodeErr() {
	execCmd := suite.NewExecCommand()
	suite.cancelFunc()
	execCmd.SetExitCodeErr(fmt.Errorf("an error"))
	select {
	case <-execCmd.exitCodeErrCh:
		suite.Fail("SetExitCodeErr set the exit code error on a cancelled context")
	default:
		// Exit code error was not sent, so the test passed
	}
}

func (suite *ExecCommandImplTestSuite) TestSetExitCodeErr_SetsExitCodeErr() {
	execCmd := suite.NewExecCommand()
	expectedErr := fmt.Errorf("error")
	execCmd.SetExitCodeErr(expectedErr)
	select {
	case err := <-execCmd.exitCodeErrCh:
		suite.Equal(expectedErr, err)
	default:
		suite.Fail("SetExitCodeErr did not send the exit code error on exitCodeErrCh")
	}
}

func (suite *ExecCommandImplTestSuite) TestExitCode_CancelledContext_ReturnsError() {
	execCmd := suite.NewExecCommand()
	suite.cancelFunc()
	_, err := execCmd.ExitCode()
	suite.Regexp(suite.ctx.Err(), err)
}

func (suite *ExecCommandImplTestSuite) TestExitCode_ReturnsExitCodeIfSet() {
	execCmd := suite.NewExecCommand()
	execCmd.SetExitCode(1)
	exitCode, err := execCmd.ExitCode()
	if suite.NoError(err) {
		suite.Equal(1, exitCode)
	}
}

func (suite *ExecCommandImplTestSuite) TestExitCode_ReturnsExitCodeErrIfSet() {
	execCmd := suite.NewExecCommand()
	expectedErr := fmt.Errorf("error")
	execCmd.SetExitCodeErr(expectedErr)
	_, err := execCmd.ExitCode()
	suite.Equal(expectedErr, err)
}

func TestExecCommandImpl(t *testing.T) {
	suite.Run(t, new(ExecCommandImplTestSuite))
}


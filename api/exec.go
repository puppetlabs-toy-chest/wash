package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

func streamCommandOutput(
	ctx context.Context,
	respStreamer chan<- *streamedJSONRespObj,
	streamName string,
	stream io.Reader,
) <-chan struct{} {
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)
		defer func() {
			if closer, ok := stream.(io.Closer); ok {
				closer.Close()
			}
		}()

		buf := make([]byte, 4096, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stream.Read(buf)
				timestamp := time.Now()

				if n < 0 {
					log.Debugf("Negative value returned when streaming %v: %v", streamName, err)
					return
				}

				if err != nil && err != io.EOF {
					log.Debugf("Errored streaming %v: %v", streamName, err)
					return
				}
				// n >= 0, err == nil or io.EOF

				if n == 0 {
					return
				}
				// n > 0, err == nil or io.EOF

				respStreamer <- newStreamedJSONRespObj(streamName, timestamp, string(buf[:n]))

				if err == io.EOF {
					return
				}
			}
		}
	}()

	return doneCh
}

func streamExitCode(
	ctx context.Context,
	respStreamer chan<- *streamedJSONRespObj,
	stdoutDone <-chan struct{},
	stderrDone <-chan struct{},
	exitCodeCB func() (int, error),
) <-chan struct{} {
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		// Wait for the context to be cancelled, or for the output streams to
		// reach EOF/error
		select {
		case <-ctx.Done():
			return
		case <-stdoutDone:
			if stderrDone != nil {
				select {
				case <-ctx.Done():
					return
				case <-stderrDone:
				}
			}

			exitCode, err := exitCodeCB()
			timestamp := time.Now()
			if err != nil {
				log.Warnf("Could not get the exit code: %v", err)
				return
			}

			respStreamer <- newStreamedJSONRespObj("exitCode", timestamp, exitCode)
		}
	}()

	return doneCh
}

type execBody struct {
	Cmd  string             `json:"cmd"`
	Args []string           `json:"args"`
	Opts plugin.ExecOptions `json:"opts"`
}

// TODO:
//   * Make streaming optional?  Making it optional would give us the benefit
//     of being able to separate stdout + stderr _and_ capture the order the streams
//     were printed in.
//		   * Running a bunch of execs in parallel would still work as-is, we'd just need
//		     to lump stdout + stderr together
//		   		* ^ Would be good to add an option to ExecOptions to turn off stderr/only return
//           	  a single stream
//
//   * Start adding events? If we're streaming JSON, events would probably be useful,
//     especially if an error happens.
//
//   * Introduce a Timestamped Reader? This is a reader that also returns the timestamp
//     of when the data was read.
//         * No way to do this in Docker
//
//
var execHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	if r.Method != http.MethodPost {
		return httpMethodNotSupported(r.Method, r.URL.Path, []string{http.MethodPost})
	}

	path := mux.Vars(r)["path"]
	log.Infof("API: Exec %v", path)

	ctx := r.Context()

	entry, errResp := getEntryFromPath(ctx, path)
	if errResp != nil {
		return errResp
	}

	exec, ok := entry.(plugin.Execable)
	if !ok {
		return unsupportedActionResponse(path, execAction)
	}

	if r.Body == nil {
		return badRequestResponse(r.URL.Path, "Please send a JSON request body")
	}

	var body execBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return badRequestResponse(r.URL.Path, err.Error())
	}

	// TODO: This and the stream endpoint have some shared code for streaming
	// responses. That should be moved to a separate helper at some point.
	fw, ok := w.(flushableWriter)
	if !ok {
		return unknownErrorResponse(fmt.Errorf("Cannot stream %v, response handler does not support flushing", path))
	}

	result, err := exec.Exec(ctx, body.Cmd, body.Args, body.Opts)
	if err != nil {
		return erroredActionResponse(path, execAction, err.Error())
	}

	objQueue := make(chan *streamedJSONRespObj)

	var stdoutDone <-chan struct{}
	if result.Stdout != nil {
		stdoutDone = streamCommandOutput(
			ctx,
			objQueue,
			"stdout",
			result.Stdout,
		)
	}

	var stderrDone <-chan struct{}
	if result.Stderr != nil {
		stderrDone = streamCommandOutput(
			ctx,
			objQueue,
			"stderr",
			result.Stderr,
		)
	}

	go func() {
		exitCodeDone := streamExitCode(
			ctx,
			objQueue,
			stdoutDone,
			stderrDone,
			result.ExitCodeCB,
		)
		<-exitCodeDone
		close(objQueue)
	}()

	w.WriteHeader(http.StatusOK)
	fw.Flush()

	streamJSONResponse(
		ctx,
		objQueue,
		&streamableResponseWriter{fw},
	)

	return nil
}

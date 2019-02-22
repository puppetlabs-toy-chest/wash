package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

type execBody struct {
	Cmd  string             `json:"cmd"`
	Args []string           `json:"args"`
	Opts plugin.ExecOptions `json:"opts"`
}

// ExecPacketType identifies the packet type.
type ExecPacketType = string

// Enumerates packet types.
const (
	Stdout   ExecPacketType = "stdout"
	Stderr   ExecPacketType = "stderr"
	Exitcode ExecPacketType = "exitcode"
)

// ExecPacket is a single packet of results from an exec.
type ExecPacket struct {
	TypeField ExecPacketType `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      interface{}    `json:"data"`
	Err       *ErrorObj      `json:"error"`
}

func newExecPacket(typeField ExecPacketType, timestamp time.Time) *ExecPacket {
	return &ExecPacket{TypeField: typeField, Timestamp: timestamp}
}

func (p *ExecPacket) send(ctx context.Context, w *json.Encoder) {
	select {
	case <-ctx.Done():
		// Don't send anything if the context's finished. Otherwise, the Encode
		// will error w/ a broken pipe.
	default:
		if err := w.Encode(p); err != nil {
			log.Warnf("Error encoding the packet from %v: %v", p.TypeField, err)
		}
	}
}

var outputStreamNames = [2]string{Stdout, Stderr}

func streamOutput(ctx context.Context, w *json.Encoder, outputCh <-chan plugin.ExecOutputChunk) {
	if outputCh == nil {
		return
	}

	for chunk := range outputCh {
		stream := outputStreamNames[chunk.StreamID]

		packet := newExecPacket(stream, chunk.Timestamp)
		if err := chunk.Err; err != nil {
			packet.Err = newStreamingErrorObj(err.Error())
		} else {
			packet.Data = chunk.Data
		}

		packet.send(ctx, w)
	}
}

func streamExitCode(ctx context.Context, w *json.Encoder, exitCodeCB func() (int, error)) {
	if exitCodeCB == nil {
		return
	}

	packet := newExecPacket(Exitcode, time.Now())

	exitCode, err := exitCodeCB()
	if err != nil {
		packet.Err = newUnknownErrorObj(err)
	} else {
		packet.Data = exitCode
	}

	packet.send(ctx, w)
}

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

	w.WriteHeader(http.StatusOK)
	fw.Flush()

	enc := json.NewEncoder(&streamableResponseWriter{fw})
	streamOutput(ctx, enc, result.OutputCh)
	streamExitCode(ctx, enc, result.ExitCodeCB)

	return nil
}

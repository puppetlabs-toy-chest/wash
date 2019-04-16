package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"
)

// Send serializes an ExecPacket via the provided json encoder.
// Skips if the provided context has been cancelled.
func sendPacket(ctx context.Context, w *json.Encoder, p *apitypes.ExecPacket) {
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

var outputStreamNames = [2]string{apitypes.Stdout, apitypes.Stderr}

func streamOutput(ctx context.Context, w *json.Encoder, outputCh <-chan plugin.ExecOutputChunk) {
	if outputCh == nil {
		return
	}

	for chunk := range outputCh {
		stream := outputStreamNames[chunk.StreamID]

		packet := apitypes.ExecPacket{TypeField: stream, Timestamp: chunk.Timestamp}
		if err := chunk.Err; err != nil {
			packet.Err = newStreamingErrorObj(stream, err.Error())
		} else {
			packet.Data = chunk.Data
		}

		sendPacket(ctx, w, &packet)
	}
}

func streamExitCode(ctx context.Context, w *json.Encoder, exitCodeCB func() (int, error)) {
	if exitCodeCB == nil {
		return
	}

	packet := apitypes.ExecPacket{TypeField: apitypes.Exitcode, Timestamp: time.Now()}

	exitCode, err := exitCodeCB()
	if err != nil {
		packet.Err = newUnknownErrorObj(fmt.Errorf("could not get the exit code: %v", err))
	} else {
		packet.Data = exitCode
	}

	sendPacket(ctx, w, &packet)
}

// swagger:route POST /fs/exec exec executeCommand
//
// Execute a command on a remote system
//
// Executes a command on the remote system described by the supplied path.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Responses:
//       200: ExecPacket
//       400: errorResp
//       404: errorResp
//       500: errorResp
var execHandler handler = func(w http.ResponseWriter, r *http.Request) *errorResponse {
	ctx := r.Context()
	entry, path, errResp := getEntryFromRequest(ctx, r)
	if errResp != nil {
		return errResp
	}

	if !plugin.ExecAction.IsSupportedOn(entry) {
		return unsupportedActionResponse(path, plugin.ExecAction)
	}

	if r.Body == nil {
		return badRequestResponse(path, plugin.ExecAction, "Please send a JSON request body")
	}

	var body apitypes.ExecBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return badRequestResponse(path, plugin.ExecAction, err.Error())
	}

	fw, ok := w.(flushableWriter)
	if !ok {
		return unknownErrorResponse(fmt.Errorf("Cannot stream %v, response handler does not support flushing", path))
	}

	activity.Record(ctx, "API: Exec %v %+v", path, body)
	opts := plugin.ExecOptions{}
	if body.Opts.Input != "" {
		opts.Stdin = strings.NewReader(body.Opts.Input)
	}
	result, err := entry.(plugin.Execable).Exec(ctx, body.Cmd, body.Args, opts)
	if err != nil {
		activity.Record(ctx, "API: Exec %v errored: %v", path, err)
		return erroredActionResponse(path, plugin.ExecAction, err.Error())
	}

	w.WriteHeader(http.StatusOK)
	fw.Flush()

	enc := json.NewEncoder(&streamableResponseWriter{fw})
	streamOutput(ctx, enc, result.OutputCh)
	streamExitCode(ctx, enc, result.ExitCodeCB)

	activity.Record(ctx, "API: Exec %v complete", path)
	return nil
}

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// TODO: This file should be better named

// Inspired by Docker's WriteFlusher: https://github.com/moby/moby/blob/17.05.x/pkg/ioutils/writeflusher.go
type flushableWriter interface {
	io.Writer
	http.Flusher
}
type streamableResponseWriter struct {
	flushableWriter
}

// Write flushes the data immediately after every write operation
func (wf *streamableResponseWriter) Write(b []byte) (n int, err error) {
	n, err = wf.flushableWriter.Write(b)
	wf.Flush()
	return n, err
}

type streamedJSONRespObj struct {
	TypeField string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Obj       interface{} `json:"obj"`
}

func newStreamedJSONRespObj(typeField string, timestamp time.Time, obj interface{}) *streamedJSONRespObj {
	return &streamedJSONRespObj{
		TypeField: typeField,
		Timestamp: timestamp,
		Obj:       obj,
	}
}

func streamJSONResponse(
	ctx context.Context,
	objQueue <-chan *streamedJSONRespObj,
	w *streamableResponseWriter,
) {
	enc := json.NewEncoder(w)

	for {
		select {
		case <-ctx.Done():
			return
		case obj, ok := <-objQueue:
			if !ok {
				// Streaming's finished, so nothing else needs to be done
				return
			}

			if err := enc.Encode(obj); err != nil {
				log.Debugf("Error encoding the object: %v", err)
			}
		}
	}
}

// Package apitypes declares types common to the API client and server.
package apitypes

import (
	"encoding/json"
	"fmt"
)

// ErrorFields represents the fields of an ErrorObj
type ErrorFields = map[string]interface{}

// ErrorObj represents an API error object
type ErrorObj struct {
	// Identifies the kind of error.
	Kind string `json:"kind"`
	// A description of what failed.
	Msg string `json:"msg"`
	// Additional structured data that may be useful in responding to the error.
	Fields ErrorFields `json:"fields"`
}

func (e *ErrorObj) Error() string {
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		// We should never hit this code-path, but better safe than sorry
		return fmt.Sprintf("Kind: %v, Msg: %v, Fields: %v", e.Kind, e.Msg, e.Fields)
	}

	return string(jsonBytes)
}

// Define error kinds returned by the API
const (
	UnsupportedAction  = "puppetlabs.wash/unsupported-action"
	UnknownError       = "puppetlabs.wash/unknown-error"
	StreamingError     = "puppetlabs.wash/streaming-error"
	EntryNotFound      = "puppetlabs.wash/entry-not-found"
	PluginDoesNotExist = "puppetlabs.wash/plugin-does-not-exist"
	BadRequest         = "puppetlabs.wash/bad-request"
	ErroredAction      = "puppetlabs.wash/errored-action"
	DuplicateCName     = "puppetlabs.wash/duplicate-cname"
	RelativePath       = "puppetlabs.wash/relative-path"
)

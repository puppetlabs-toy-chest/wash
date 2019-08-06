// Package apitypes declares types common to the API client and server.
package apitypes

import (
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

// Provides a nicely formatted string representation of the error. Omits Fields.
// If you want to use fields, either access them directly or serialize the whole thing
// to JSON as part of a larger set of JSON output.
func (e *ErrorObj) Error() string {
	return fmt.Sprintf("%v: %v", e.Kind, e.Msg)
}

// Define error kinds returned by the API
const (
	UnsupportedAction  = "puppetlabs.wash/unsupported-action"
	UnknownError       = "puppetlabs.wash/unknown-error"
	StreamingError     = "puppetlabs.wash/streaming-error"
	EntryNotFound      = "puppetlabs.wash/entry-not-found"
	PluginDoesNotExist = "puppetlabs.wash/plugin-does-not-exist"
	BadRequest         = "puppetlabs.wash/bad-request"
	BadActionRequest   = "puppetlabs.wash/bad-action-request"
	JournalUnavailable = "puppetlabs.wash/journal-unavailable"
	ErroredAction      = "puppetlabs.wash/errored-action"
	DuplicateCName     = "puppetlabs.wash/duplicate-cname"
	RelativePath       = "puppetlabs.wash/relative-path"
	InvalidPaths       = "puppetlabs.wash/invalid-paths"
	OutOfBounds        = "puppetlabs.wash/out-of-bounds"
	NonWashPath        = "puppetlabs.wash/non-wash-path"
	InvalidBool        = "puppetlabs.wash/invalid-bool"
)

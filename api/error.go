package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// This approach was adapted from https://blog.golang.org/error-handling-and-go

// ErrorFields represents the fields of an ErrorObj
type ErrorFields = map[string]interface{}

// ErrorObj represents an API error object
type ErrorObj struct {
	Kind   string      `json:"kind"`
	Msg    string      `json:"msg"`
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

func newAPIErrorObj(kind string, message string, fields ErrorFields) ErrorObj {
	return ErrorObj{
		Kind:   "puppetlabs.wash/" + kind,
		Msg:    message,
		Fields: fields,
	}
}

// ErrorResponse represents an error response
type errorResponse struct {
	statusCode int
	body       ErrorObj
}

func (e *errorResponse) Error() string {
	return e.body.Error()
}

// Below are all of Wash's API error responses

func unknownErrorResponse(err error) *errorResponse {
	statusCode := http.StatusInternalServerError
	body := newAPIErrorObj(
		"unknown-error",
		err.Error(),
		ErrorFields{},
	)

	return &errorResponse{statusCode, body}
}

func entryNotFoundResponse(path string, reason string) *errorResponse {
	fields := ErrorFields{"path": path}

	statusCode := http.StatusNotFound
	body := newAPIErrorObj(
		"entry-not-found",
		fmt.Sprintf("Could not find entry %v: %v", path, reason),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func pluginDoesNotExistResponse(plugin string) *errorResponse {
	fields := ErrorFields{"plugin": plugin}

	statusCode := http.StatusNotFound
	body := newAPIErrorObj(
		"plugin-does-not-exist",
		fmt.Sprintf("Plugin %v does not exist", plugin),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func unsupportedActionResponse(path string, action *action) *errorResponse {
	fields := ErrorFields{
		"path":   path,
		"action": action,
	}

	statusCode := http.StatusNotFound
	msg := fmt.Sprintf("Entry %v does not support the %v action: It does not implement the %v protocol", path, action.Name, action.Protocol)
	body := newAPIErrorObj(
		"unsupported-action",
		msg,
		fields,
	)

	return &errorResponse{statusCode, body}
}

func erroredActionResponse(path string, action *action, reason string) *errorResponse {
	fields := ErrorFields{
		"path":   path,
		"action": action.Name,
	}

	statusCode := http.StatusInternalServerError
	body := newAPIErrorObj(
		"errored-action",
		fmt.Sprintf("The %v action errored on %v: %v", action.Name, path, reason),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func httpMethodNotSupported(method string, path string, supported []string) *errorResponse {
	fields := ErrorFields{
		"method":    method,
		"path":      path,
		"supported": supported,
	}

	body := newAPIErrorObj(
		"http-method-not-supported",
		fmt.Sprintf("The %v method is not supported for %v, supported methods are: %v", method, path, strings.Join(supported, ", ")),
		fields,
	)

	return &errorResponse{http.StatusNotFound, body}
}

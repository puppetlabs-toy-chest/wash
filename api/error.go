package api

import (
	"fmt"
	"net/http"
	"strings"

	apitypes "github.com/puppetlabs/wash/api/types"
)

// This approach was adapted from https://blog.golang.org/error-handling-and-go

func newErrorObj(kind string, message string, fields apitypes.ErrorFields) *apitypes.ErrorObj {
	return &apitypes.ErrorObj{
		Kind:   "puppetlabs.wash/" + kind,
		Msg:    message,
		Fields: fields,
	}
}

func newUnknownErrorObj(err error) *apitypes.ErrorObj {
	return newErrorObj("unknown-error", err.Error(), apitypes.ErrorFields{})
}

func newStreamingErrorObj(reason string) *apitypes.ErrorObj {
	return newErrorObj("streaming-error", reason, apitypes.ErrorFields{})
}

// ErrorResponse represents an error response
type errorResponse struct {
	statusCode int
	body       *apitypes.ErrorObj
}

func (e *errorResponse) Error() string {
	return e.body.Error()
}

// Below are all of Wash's API error responses

func unknownErrorResponse(err error) *errorResponse {
	statusCode := http.StatusInternalServerError
	body := newUnknownErrorObj(err)

	return &errorResponse{statusCode, body}
}

func entryNotFoundResponse(path string, reason string) *errorResponse {
	fields := apitypes.ErrorFields{"path": path}

	statusCode := http.StatusNotFound
	body := newErrorObj(
		"entry-not-found",
		fmt.Sprintf("Could not find entry %v: %v", path, reason),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func pluginDoesNotExistResponse(plugin string) *errorResponse {
	fields := apitypes.ErrorFields{"plugin": plugin}

	statusCode := http.StatusNotFound
	body := newErrorObj(
		"plugin-does-not-exist",
		fmt.Sprintf("Plugin %v does not exist", plugin),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func unsupportedActionResponse(path string, action *action) *errorResponse {
	fields := apitypes.ErrorFields{
		"path":   path,
		"action": action,
	}

	statusCode := http.StatusNotFound
	msg := fmt.Sprintf("Entry %v does not support the %v action: It does not implement the %v protocol", path, action.Name, action.Protocol)
	body := newErrorObj(
		"unsupported-action",
		msg,
		fields,
	)

	return &errorResponse{statusCode, body}
}

func badRequestResponse(path string, reason string) *errorResponse {
	fields := apitypes.ErrorFields{"path": path}
	body := newErrorObj(
		"bad-request",
		fmt.Sprintf("Bad request on %v: %v", path, reason),
		fields,
	)
	return &errorResponse{http.StatusBadRequest, body}
}

func erroredActionResponse(path string, action *action, reason string) *errorResponse {
	fields := apitypes.ErrorFields{
		"path":   path,
		"action": action.Name,
	}

	statusCode := http.StatusInternalServerError
	body := newErrorObj(
		"errored-action",
		fmt.Sprintf("The %v action errored on %v: %v", action.Name, path, reason),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func httpMethodNotSupported(method string, path string, supported []string) *errorResponse {
	fields := apitypes.ErrorFields{
		"method":    method,
		"path":      path,
		"supported": supported,
	}

	body := newErrorObj(
		"http-method-not-supported",
		fmt.Sprintf("The %v method is not supported for %v, supported methods are: %v", method, path, strings.Join(supported, ", ")),
		fields,
	)

	return &errorResponse{http.StatusNotFound, body}
}

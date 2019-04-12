package api

import (
	"fmt"
	"net/http"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
)

// This approach was adapted from https://blog.golang.org/error-handling-and-go

// swagger:response
//nolint:deadcode,unused
type errorResp struct {
	Body struct {
		apitypes.ErrorObj
	}
}

func newErrorObj(kind string, message string, fields apitypes.ErrorFields) *apitypes.ErrorObj {
	return &apitypes.ErrorObj{
		Kind:   kind,
		Msg:    message,
		Fields: fields,
	}
}

func newUnknownErrorObj(err error) *apitypes.ErrorObj {
	return newErrorObj(apitypes.UnknownError, err.Error(), apitypes.ErrorFields{})
}

func newStreamingErrorObj(stream string, reason string) *apitypes.ErrorObj {
	return newErrorObj(
		apitypes.StreamingError,
		fmt.Sprintf("error streaming %v: %v", stream, reason),
		apitypes.ErrorFields{"stream": stream},
	)
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
		apitypes.EntryNotFound,
		fmt.Sprintf("Could not find entry %v: %v", path, reason),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func pluginDoesNotExistResponse(plugin string) *errorResponse {
	fields := apitypes.ErrorFields{"plugin": plugin}

	statusCode := http.StatusNotFound
	body := newErrorObj(
		apitypes.PluginDoesNotExist,
		fmt.Sprintf("Plugin %v does not exist", plugin),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func unsupportedActionResponse(path string, a plugin.Action) *errorResponse {
	fields := apitypes.ErrorFields{
		"path":   path,
		"action": a,
	}

	statusCode := http.StatusNotFound
	msg := fmt.Sprintf("Entry %v does not support the %v action: It does not implement the %v protocol", path, a.Name, a.Protocol)
	body := newErrorObj(
		apitypes.UnsupportedAction,
		msg,
		fields,
	)

	return &errorResponse{statusCode, body}
}

func badRequestResponse(path string, a plugin.Action, reason string) *errorResponse {
	fields := apitypes.ErrorFields{
		"path":   path,
		"action": a.Name,
	}
	body := newErrorObj(
		apitypes.BadRequest,
		fmt.Sprintf("Bad request for %v action on %v: %v", a.Name, path, reason),
		fields,
	)
	return &errorResponse{http.StatusBadRequest, body}
}

func erroredActionResponse(path string, a plugin.Action, reason string) *errorResponse {
	fields := apitypes.ErrorFields{
		"path":   path,
		"action": a.Name,
	}

	statusCode := http.StatusInternalServerError
	body := newErrorObj(
		apitypes.ErroredAction,
		fmt.Sprintf("The %v action errored on %v: %v", a.Name, path, reason),
		fields,
	)

	return &errorResponse{statusCode, body}
}

func duplicateCNameResponse(e plugin.DuplicateCNameErr) *errorResponse {
	fields := apitypes.ErrorFields{
		"parent_id":                           e.ParentID,
		"first_child_name":                    e.FirstChildName,
		"first_child_slash_replacement_char":  e.FirstChildSlashReplacementChar,
		"second_child_name":                   e.SecondChildName,
		"second_child_slash_replacement_char": e.SecondChildSlashReplacementChar,
		"cname":                               e.CName,
	}

	body := newErrorObj(
		apitypes.DuplicateCName,
		e.Error(),
		fields,
	)

	return &errorResponse{http.StatusInternalServerError, body}
}

func relativePathResponse(path string) *errorResponse {
	fields := apitypes.ErrorFields{
		"path": path,
	}

	body := newErrorObj(
		apitypes.RelativePath,
		fmt.Sprintf("%v is a relative path. The Wash API only accepts absolute paths.", path),
		fields,
	)

	return &errorResponse{http.StatusBadRequest, body}
}

func nonWashEntryResponse(path string) *errorResponse {
	fields := apitypes.ErrorFields{
		"path": path,
	}

	body := newErrorObj(
		apitypes.NonWashEntry,
		fmt.Sprintf("%v is not a Wash entry.", path),
		fields,
	)

	return &errorResponse{http.StatusBadRequest, body}
}

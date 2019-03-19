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

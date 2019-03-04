package apitypes

import (
	"github.com/puppetlabs/wash/plugin"
)

// ListEntry represents a single entry from the result of issuing a wash "list"
// request.
type ListEntry struct {
	Actions    []string             `json:"actions"`
	Name       string               `json:"name"`
	Attributes plugin.Attributes    `json:"attributes"`
	Errors     map[string]*ErrorObj `json:"errors"`
}

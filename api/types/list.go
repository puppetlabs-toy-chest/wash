package apitypes

import (
	"github.com/puppetlabs/wash/plugin"
)

// ListEntry represents a single entry from the result of issuing a wash 'list' request.
//
// swagger:response
type ListEntry struct {
	Path       string               `json:"path"`
	Actions    []string             `json:"actions"`
	Name       string               `json:"name"`
	CName      string               `json:"cname"`
	Attributes plugin.Attributes    `json:"attributes"`
	Errors     map[string]*ErrorObj `json:"errors"`
}

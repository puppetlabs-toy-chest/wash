package apitypes

import "github.com/puppetlabs/wash/plugin"

// Entry represents a Wash entry as interpreted by the API.
//
// swagger:response
type Entry struct {
	Path       string                 `json:"path"`
	Actions    []string               `json:"actions"`
	Name       string                 `json:"name"`
	CName      string                 `json:"cname"`
	Attributes plugin.EntryAttributes `json:"attributes"`
}

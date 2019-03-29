package apitypes

import (
	"os"
	"time"
)

// EntryAttributes represents an entry's attributes.
//
// TODO: Add meta later
type EntryAttributes struct {
	Atime *time.Time   `json:"atime,omitempty"`
	Mtime *time.Time   `json:"mtime,omitempty"`
	Ctime *time.Time   `json:"ctime,omitempty"`
	Mode  *os.FileMode `json:"mode,omitempty"`
	Size  *uint64      `json:"size,omitempty"`
}

// ListEntry represents a single entry from the result of issuing a wash 'list' request.
//
// swagger:response
type ListEntry struct {
	Path       string               `json:"path"`
	Actions    []string             `json:"actions"`
	Name       string               `json:"name"`
	CName      string               `json:"cname"`
	Attributes EntryAttributes      `json:"attributes"`
	Errors     map[string]*ErrorObj `json:"errors"`
}

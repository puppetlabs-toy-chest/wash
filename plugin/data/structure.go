package data

import (
	"context"
	"encoding/json"

	"github.com/puppetlabs/wash/plugin"
)

// Entry is the representation of a resource.
//
// Name can be omitted for all but the root entry. The act of listing entries will fill 'name'.
//
// The presence of 'enum' or 'list' enables LS(), and either the names of child entries or how to
// query them. 'list' must produce a list of names.
//
// If 'post' and 'list' are both present, 'list' must instead produce JSON data, and 'post' will be
// used to process the result of executing 'list'. 'list' can be templated to access fields of the
// entry, and 'post' can be templated to access the resulting structured data from 'list'.
//
// The 'entry' field can be used to populate the child entries produced from 'enum' or 'list'.
//
// Metadata specifies how to retrieve metadata (if provided).
type entry struct {
	Label  string `yaml:"name"`
	Parent plugin.Entry

	Enum  []string `yaml:"enum,omitempty"`
	List  string   `yaml:"list"`
	Post  string   `yaml:"post"`
	Proto *entry   `yaml:"proto"`

	Meta string `yaml:"metadata"`
}

func (e *entry) Name() string {
	return e.Label
}

func (e *entry) CacheConfig() *plugin.CacheConfig {
	return nil
}

func (e *entry) Metadata(context.Context) (plugin.MetadataMap, error) {
	if e.Meta == "" {
		return plugin.MetadataMap{}, nil
	}

	output, err := invoke(e.Meta, e)
	if err != nil {
		return nil, err
	}

	var data plugin.MetadataMap
	dec := json.NewDecoder(output)
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func newCopy(name string, parent plugin.Entry, example *entry) plugin.Entry {
	if example != nil && (example.List != "" || example.Enum != nil) {
		newEntry := listableEntry{*example}
		newEntry.Label = name
		newEntry.Parent = parent
		return &newEntry
	}

	var newEntry entry
	if example != nil {
		newEntry = *example
	}
	newEntry.Label = name
	newEntry.Parent = parent
	return &newEntry
}

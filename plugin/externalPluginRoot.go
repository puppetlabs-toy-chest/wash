package plugin

import (
	"context"
	"encoding/json"
)

// ExternalPluginSpec represents an external plugin's YAML specification.
type ExternalPluginSpec struct {
	Name   string
	Script string
}

// ExternalPluginRoot represents an external plugin's root.
type ExternalPluginRoot struct {
	*ExternalPluginEntry
}

// NewExternalPluginRoot returns a new external plugin root given
// the plugin script
func NewExternalPluginRoot(plugin ExternalPluginSpec) *ExternalPluginRoot {
	return &ExternalPluginRoot{&ExternalPluginEntry{
		script: ExternalPluginScript{Path: plugin.Script},
	}}
}

// Init initializes the external plugin root
func (r *ExternalPluginRoot) Init() error {
	script := r.script

	stdout, err := r.script.InvokeAndWait("init")
	if err != nil {
		return err
	}

	var decodedRoot decodedExternalPluginEntry
	if err := json.Unmarshal(stdout, &decodedRoot); err != nil {
		return err
	}

	entry, err := decodedRoot.toExternalPluginEntry()
	if err != nil {
		return err
	}

	r.ExternalPluginEntry = entry
	r.ExternalPluginEntry.script = script
	r.ExternalPluginEntry.washPath = "/" + r.Name()

	return nil
}

// LS lists the root's entries.
//
// We need this b/c *ExternalPluginEntry#LS has a different receiver
// (*ExternalPluginEntry) than *ExternalPluginRoot (i.e. b/c Go's
// typechecker complains about it)
func (r *ExternalPluginRoot) LS(ctx context.Context) ([]Entry, error) {
	return r.ExternalPluginEntry.LS(ctx)
}

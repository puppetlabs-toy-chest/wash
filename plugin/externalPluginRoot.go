package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/journal"
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
		script: NewExternalPluginScript(plugin.Script),
	}}
}

// Init initializes the external plugin root
func (r *ExternalPluginRoot) Init() error {
	ctx := context.Background()
	script := r.script

	stdout, err := r.script.InvokeAndWait(ctx, "init")
	if err != nil {
		return err
	}

	var decodedRoot decodedExternalPluginEntry
	if err := json.Unmarshal(stdout, &decodedRoot); err != nil {
		journal.Record(
			ctx,
			"could not decode the plugin root from stdout\nreceived:\n%v\nexpected something like:\n%v",
			strings.TrimSpace(string(stdout)),
			"{\"name\":\"<name_of_root_dir>\",\"supported_actions\":[\"list\"]}",
		)

		return fmt.Errorf("could not decode the plugin root from stdout: %v", err)
	}

	entry, err := decodedRoot.toExternalPluginEntry()
	if err != nil {
		return err
	}

	r.ExternalPluginEntry = entry
	r.ExternalPluginEntry.script = script

	return nil
}

// List lists the root's entries.
//
// We need this b/c *ExternalPluginEntry#List has a different receiver
// (*ExternalPluginEntry) than *ExternalPluginRoot (i.e. b/c Go's
// typechecker complains about it)
func (r *ExternalPluginRoot) List(ctx context.Context) ([]Entry, error) {
	return r.ExternalPluginEntry.List(ctx)
}

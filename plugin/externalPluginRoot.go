package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/journal"
)

// externalPluginRoot represents an external plugin's root.
type externalPluginRoot struct {
	*externalPluginEntry
}

// newExternalPluginRoot returns a new external plugin root given
// the plugin script
func newExternalPluginRoot(script string) *externalPluginRoot {
	return &externalPluginRoot{&externalPluginEntry{
		script: externalPluginScriptImpl{path: script},
	}}
}

// Init initializes the external plugin root
func (r *externalPluginRoot) Init() error {
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

	r.externalPluginEntry = entry
	r.externalPluginEntry.script = script

	return nil
}

// List lists the root's entries.
//
// We need this b/c *externalPluginEntry#List has a different receiver
// (*externalPluginEntry) than *externalPluginRoot (i.e. b/c Go's
// typechecker complains about it)
func (r *externalPluginRoot) List(ctx context.Context) ([]Entry, error) {
	return r.externalPluginEntry.List(ctx)
}

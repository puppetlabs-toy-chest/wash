package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/puppetlabs/wash/activity"
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
	// Give external plugins about five-seconds to finish their
	// initialization
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancelFunc()
	stdout, err := r.script.InvokeAndWait(ctx, "init", nil)
	if err != nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out while waiting for init to finish")
		default:
			return err
		}
	}
	var decodedRoot decodedExternalPluginEntry
	if err := json.Unmarshal(stdout, &decodedRoot); err != nil {
		activity.Record(
			ctx,
			"could not decode the plugin root from stdout\nreceived:\n%v\nexpected something like:\n%v",
			strings.TrimSpace(string(stdout)),
			"{\"name\":\"<name_of_root_dir>\",\"methods\":[\"list\"]}",
		)
		return fmt.Errorf("could not decode the plugin root from stdout: %v", err)
	}
	entry, err := decodedRoot.toExternalPluginEntry()
	if err != nil {
		return err
	}
	script := r.script
	r.externalPluginEntry = entry
	r.externalPluginEntry.script = script
	return nil
}

// List lists the root's entries.
//
// We need this b/c *externalPluginEntry#List has a different receiver
// (*externalPluginEntry) than *externalPluginRoot (i.e. b/c Go's
// typechecker complains about it)
//
// TODO: Is this still an issue?
func (r *externalPluginRoot) List(ctx context.Context) ([]Entry, error) {
	return r.externalPluginEntry.List(ctx)
}

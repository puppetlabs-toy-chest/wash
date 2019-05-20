package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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
		return newStdoutDecodeErr(
			nil,
			"the plugin root",
			err,
			stdout,
			"{\"name\":\"plugin_name\",\"methods\":[\"list\"]}",
		)
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
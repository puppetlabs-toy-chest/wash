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

// Init initializes the external plugin root
func (r *externalPluginRoot) Init(cfg map[string]interface{}) error {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal plugin config %v into JSON: %v", cfg, err)
	}

	// Give external plugins about five-seconds to finish their
	// initialization
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	stdout, err := r.script.InvokeAndWait(ctx, "init", nil, string(cfgJSON))
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
			context.Background(),
			"the plugin root",
			err,
			stdout,
			"{}",
		)
	}

	// Fill in required fields with data we already know.
	if decodedRoot.Name == "" {
		decodedRoot.Name = r.Name()
	} else if decodedRoot.Name != r.Name() {
		panic(fmt.Sprintf(`plugin root's name must match the basename (without extension) of %s
it's safe to omit name from the response to 'init'`, r.script.Path()))
	}
	if decodedRoot.Methods == nil {
		decodedRoot.Methods = []string{"list"}
	}
	entry, err := decodedRoot.toExternalPluginEntry()
	if err != nil {
		return err
	}
	if !ListAction().IsSupportedOn(entry) {
		panic(fmt.Sprintf("plugin root for %s must implement 'list'", r.script.Path()))
	}
	script := r.script
	r.externalPluginEntry = entry
	r.externalPluginEntry.script = script
	return nil
}

func (r *externalPluginRoot) WrappedTypes() SchemaMap {
	// This only makes sense for core plugins because it is a Go-specific
	// limitation.
	return nil
}

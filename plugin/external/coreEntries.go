package external

import (
	"encoding/json"
	"fmt"

	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/volume"
)

type coreEntry interface {
	createInstance(parent *pluginEntry, decodedEntry decodedExternalPluginEntry) (plugin.Entry, error)
	schema() *plugin.EntrySchema
}

var coreEntries = map[string]coreEntry{
	"__volume::fs__": volumeFS{},
}

type volumeFS struct{}

func (volumeFS) createInstance(parent *pluginEntry, e decodedExternalPluginEntry) (plugin.Entry, error) {
	var opts struct{ Maxdepth uint }
	// Use a default of 3 if unspecified.
	opts.Maxdepth = 3

	if err := json.Unmarshal([]byte(e.State), &opts); err != nil {
		return nil, fmt.Errorf("volume filesystem options invalid: %v", err)
	}

	return volume.NewFS(e.Name, parent, int(opts.Maxdepth)), nil
}

func (volumeFS) schema() *plugin.EntrySchema {
	return (&volume.FS{}).Schema()
}

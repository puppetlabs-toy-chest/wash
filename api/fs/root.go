package apifs

import (
	"fmt"
	"os"

	"github.com/puppetlabs/wash/plugin"
)

// Root of the local filesystem plugin
type Root struct {
	dir
}

// Init for root
func (r *Root) Init(cfg map[string]interface{}) error {
	var basepath string
	if pathI, ok := cfg["basepath"]; ok {
		basepath, ok = pathI.(string)
		if !ok {
			return fmt.Errorf("local.basepath config must be a string, not %s", pathI)
		}
	} else {
		basepath = "/tmp"
	}

	finfo, err := os.Stat(basepath)
	if err != nil {
		return err
	}

	r.dir.fsnode = newFSNode(finfo, basepath)

	r.EntryBase = plugin.NewEntry("local")
	r.DisableDefaultCaching()
	return nil
}

// Schema returns the root's schema
func (r *Root) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(r, "local").
		SetDescription(rootDescription).
		IsSingleton()
}

var _ = plugin.Root(&Root{})

const rootDescription = `
This plugin exposes part of the local filesystem. The path is mounted based on the path specified
in the WASH_LOCALFS environment variable.
`

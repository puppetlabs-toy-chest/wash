package external

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/wash/plugin"
)

// PluginSpec represents an external plugin's specification.
type PluginSpec struct {
	Script string
}

// Name returns the plugin name, which is the basename of the script with extension removed.
func (s PluginSpec) Name() string {
	basename := filepath.Base(s.Script)
	return strings.TrimSuffix(basename, filepath.Ext(basename))
}

// Load ensures the external plugin represents an executable artifact and create a plugin Root.
func (s PluginSpec) Load() (plugin.Root, error) {
	fi, err := os.Stat(s.Script)
	if err != nil {
		return nil, err
	} else if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("script %v is not a file", s.Script)
	} else if fi.Mode().Perm()&0100 == 0 {
		return nil, fmt.Errorf("script %v is not executable", s.Script)
	}

	root := &pluginRoot{pluginEntry{
		EntryBase: plugin.NewEntry(s.Name()),
		script:    externalPluginScriptImpl{path: s.Script},
	}}
	return root, nil
}

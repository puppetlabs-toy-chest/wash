package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExternalPluginSpec represents an external plugin's specification.
type ExternalPluginSpec struct {
	Script string
}

// Name returns the plugin name, which is the basename of the script with extension removed.
func (s ExternalPluginSpec) Name() string {
	basename := filepath.Base(s.Script)
	return strings.TrimSuffix(basename, filepath.Ext(basename))
}

// Load ensures the external plugin represents an executable artifact and create a plugin Root.
func (s ExternalPluginSpec) Load() (Root, error) {
	fi, err := os.Stat(s.Script)
	if err != nil {
		return nil, err
	} else if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("script %v is not a file", s.Script)
	} else if fi.Mode().Perm()&0100 == 0 {
		return nil, fmt.Errorf("script %v is not executable", s.Script)
	}

	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry(s.Name()),
		script:    externalPluginScriptImpl{path: s.Script},
	}}
	return root, nil
}

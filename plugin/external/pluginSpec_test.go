package external

import (
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

func TestLoadExternalPlugin(t *testing.T) {
	spec := PluginSpec{Script: "testdata/external.sh"}
	root, err := spec.Load()
	assert.NoError(t, err)
	assert.Equal(t, "external", plugin.Name(root))
}

func TestLoadExternalPluginNoExec(t *testing.T) {
	spec := PluginSpec{Script: "testdata/noexec"}
	_, err := spec.Load()
	assert.EqualError(t, err, "script testdata/noexec is not executable")
}

func TestRegisterExternalPluginNoExist(t *testing.T) {
	spec := PluginSpec{Script: "testdata/noexist"}
	_, err := spec.Load()
	assert.EqualError(t, err, "stat testdata/noexist: no such file or directory")
}

func TestRegisterExternalPluginNotFile(t *testing.T) {
	spec := PluginSpec{Script: "testdata/notfile"}
	_, err := spec.Load()
	assert.EqualError(t, err, "script testdata/notfile is not a file")
}

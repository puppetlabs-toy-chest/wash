package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadExternalPlugin(t *testing.T) {
	spec := ExternalPluginSpec{Script: "testdata/external.sh"}
	root, err := spec.Load()
	assert.NoError(t, err)
	assert.Equal(t, "external", root.name())
}

func TestLoadExternalPluginNoExec(t *testing.T) {
	spec := ExternalPluginSpec{Script: "testdata/noexec"}
	_, err := spec.Load()
	assert.EqualError(t, err, "script testdata/noexec is not executable")
}

func TestRegisterExternalPluginNoExist(t *testing.T) {
	spec := ExternalPluginSpec{Script: "testdata/noexist"}
	_, err := spec.Load()
	assert.EqualError(t, err, "stat testdata/noexist: no such file or directory")
}

func TestRegisterExternalPluginNotFile(t *testing.T) {
	spec := ExternalPluginSpec{Script: "testdata/notfile"}
	_, err := spec.Load()
	assert.EqualError(t, err, "script testdata/notfile is not a file")
}

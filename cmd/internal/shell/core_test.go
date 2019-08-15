package shell

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	oldShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", oldShell)

	os.Setenv("SHELL", "")
	sh := Get()
	assert.IsType(t, basic{}, sh)

	os.Setenv("SHELL", "zsh")
	sh = Get()
	assert.IsType(t, zsh{}, sh)

	os.Setenv("SHELL", "/bin/zsh")
	sh = Get()
	assert.IsType(t, zsh{}, sh)

	os.Setenv("SHELL", "bash")
	sh = Get()
	assert.IsType(t, bash{}, sh)

	os.Setenv("SHELL", "/usr/local/bin/bash")
	sh = Get()
	assert.IsType(t, bash{}, sh)
}

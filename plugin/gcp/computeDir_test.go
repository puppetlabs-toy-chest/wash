package gcp

import (
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

func TestComputeDir(t *testing.T) {
	dir, err := newComputeDir(nil, "dummy")
	if assert.NoError(t, err) {
		assert.Implements(t, (*plugin.Parent)(nil), dir)
	}
}

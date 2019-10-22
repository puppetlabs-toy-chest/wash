package gcp

import (
	"testing"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

func TestComputeDir(t *testing.T) {
	ctx := plugin.SetTestCache(datastore.NewMemCache())
	defer plugin.UnsetTestCache()

	dir, err := newComputeDir(ctx, nil, "dummy")
	if assert.NoError(t, err) {
		assert.Implements(t, (*plugin.Parent)(nil), dir)
	}
}

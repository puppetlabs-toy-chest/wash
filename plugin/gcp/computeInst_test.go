package gcp

import (
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
	compute "google.golang.org/api/compute/v1"
)

func TestComputeInstance(t *testing.T) {
	inst := compute.Instance{Name: "foo"}
	compInst := newComputeInstance(&inst, computeProjectService{})
	assert.Equal(t, "foo", compInst.Name())
	assert.Implements(t, (*plugin.Parent)(nil), compInst)
	assert.Implements(t, (*plugin.Execable)(nil), compInst)
}

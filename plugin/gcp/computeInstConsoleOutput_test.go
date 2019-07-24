package gcp

import (
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
	compute "google.golang.org/api/compute/v1"
)

func TestComputeInstanceConsoleOutput(t *testing.T) {
	inst := compute.Instance{Name: "foo"}
	outInst := newComputeInstanceConsoleOutput(&inst, computeProjectService{})
	assert.Equal(t, "console.out", outInst.Name())
	assert.Implements(t, (*plugin.Readable)(nil), outInst)
}

package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	sh := Get()
	assert.IsType(t, basic{}, sh)
}

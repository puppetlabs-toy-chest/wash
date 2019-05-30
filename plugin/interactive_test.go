package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInteractive(t *testing.T) {
	if !IsInteractive() {
		// If tests are not run interactively we can only test the false case, so override to true.
		isInteractive = true
	}

	InitInteractive(true)
	assert.True(t, IsInteractive())

	InitInteractive(false)
	assert.False(t, IsInteractive())

	InitInteractive(true)
	assert.False(t, IsInteractive())
}

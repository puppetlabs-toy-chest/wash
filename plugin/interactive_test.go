package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInteractive(t *testing.T) {
	saveInteractive := isInteractive
	defer func() { isInteractive = saveInteractive }()

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

func TestWithConsole(t *testing.T) {
	saveInteractive := isInteractive
	defer func() { isInteractive = saveInteractive }()

	called := false
	passing := func(context.Context) error { called = true; return nil }
	failing := func(context.Context) error { called = true; return fmt.Errorf("failed") }

	// Test non-interactive first.
	isInteractive = false
	assert.NoError(t, withConsole(context.Background(), passing))
	assert.True(t, called)

	called = false
	assert.Error(t, withConsole(context.Background(), failing), "failed")
	assert.True(t, called)

	// Interactive tests.
	isInteractive = true

	called = false
	if saveInteractive {
		assert.NoError(t, withConsole(context.Background(), passing))
		assert.True(t, called)
	} else {
		assert.Error(t, withConsole(context.Background(), passing))
		assert.False(t, called)
	}
}

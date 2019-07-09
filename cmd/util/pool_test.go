package cmdutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that a pool with a single worker finishes.
func TestPool1(t *testing.T) {
	p := NewPool(1)

	val := 0
	p.Submit(func() {
		val++
		p.Done()
	})

	p.Finish()
	assert.Equal(t, 1, val)
}

// Test that a pool with two workers executes them concurrently and finishes.
func TestPool2(t *testing.T) {
	p := NewPool(2)

	var mux1, mux2 sync.Mutex
	val := 0
	// Start with both mutexes locked. In sequence wait on one and unlock the other so that both
	// functions must run concurrently to correctly unlock them.
	mux1.Lock()
	mux2.Lock()
	p.Submit(func() {
		// Wait on 1.
		mux1.Lock()
		val++
		// Signal 2.
		mux2.Unlock()
		p.Done()
	})

	p.Submit(func() {
		// Signal 1.
		mux1.Unlock()
		// Wait on 2.
		mux2.Lock()
		val++
		p.Done()
	})
	// At the end both mutexes are again locked.

	// Wait for completion and ensure both functions have updated the value.
	p.Finish()
	assert.Equal(t, 2, val)
}

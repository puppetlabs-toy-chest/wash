package plugin

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanupReader(t *testing.T) {
	called := false
	cleanup := func() {
		called = true
	}

	r, w := io.Pipe()
	defer w.Close()
	rdr := CleanupReader{ReadCloser: r, Cleanup: cleanup}
	go func() {
		_, err := w.Write([]byte("hello"))
		assert.NoError(t, err)
	}()

	buf := make([]byte, 5)
	n, err := rdr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(buf))
	assert.False(t, called)

	rdr.Close()
	assert.True(t, called)
	_, err = w.Write([]byte("goodbye"))
	assert.Error(t, err)
}

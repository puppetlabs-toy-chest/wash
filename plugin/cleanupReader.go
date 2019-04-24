package plugin

import "io"

// CleanupReader is a wrapper for an io.ReadCloser that performs cleanup when closed.
type CleanupReader struct {
	io.ReadCloser
	Cleanup func()
}

// Close closes the reader it wraps, then calls the Cleanup function and returns any errors.
func (c CleanupReader) Close() error {
	err := c.ReadCloser.Close()
	c.Cleanup()
	return err
}

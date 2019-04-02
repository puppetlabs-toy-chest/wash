package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// ContentCB accepts a path and returns the content associated with that path.
type ContentCB = func(context.Context, string) (plugin.SizedReader, error)

// File represents a file in a volume that has content we can access.
type File struct {
	plugin.EntryBase
	contentcb ContentCB
	path      string
}

// NewFile creates a VolumeFile.
func NewFile(name string, attr plugin.EntryAttributes, cb ContentCB, path string) *File {
	vf := &File{
		EntryBase: plugin.NewEntry(name),
		contentcb: cb,
		path:      path,
	}
	vf.SetAttributes(attr)
	vf.SetTTLOf(plugin.OpenOp, 60*time.Second)

	return vf
}

// Open returns the content of the file as a SizedReader.
func (v *File) Open(ctx context.Context) (plugin.SizedReader, error) {
	return v.contentcb(ctx, v.path)
}

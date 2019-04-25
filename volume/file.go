package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// contentCB accepts a path and returns the content associated with that path.
type contentCB = func(context.Context, string) (plugin.SizedReader, error)

// file represents a file in a volume that has content we can access.
type file struct {
	plugin.EntryBase
	contentcb contentCB
	path      string
}

// newFile creates a VolumeFile.
func newFile(name string, attr plugin.EntryAttributes, cb contentCB, path string) *file {
	vf := &file{
		EntryBase: plugin.NewEntry(name),
		contentcb: cb,
		path:      path,
	}
	vf.SetAttributes(attr)
	vf.SetTTLOf(plugin.OpenOp, 60*time.Second)

	return vf
}

// Open returns the content of the file as a SizedReader.
func (v *file) Open(ctx context.Context) (plugin.SizedReader, error) {
	return v.contentcb(ctx, v.path)
}

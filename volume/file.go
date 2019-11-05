package volume

import (
	"context"
	"io"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// file represents a file in a volume that has content we can access.
type file struct {
	plugin.EntryBase
	impl   Interface
	path   string
	dirmap *dirMap
}

// newFile creates a VolumeFile.
func newFile(name string, attr plugin.EntryAttributes, impl Interface, path string) *file {
	vf := &file{
		EntryBase: plugin.NewEntry(name),
	}
	vf.impl = impl
	vf.path = path
	vf.SetAttributes(attr)
	vf.SetTTLOf(plugin.OpenOp, 60*time.Second)

	return vf
}

func (v *file) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(v, "file").SetDescription(fileDescription)
}

// Open returns the content of the file as a SizedReader.
func (v *file) Open(ctx context.Context) (plugin.SizedReader, error) {
	return v.impl.VolumeOpen(ctx, v.path)
}

func (v *file) Stream(ctx context.Context) (io.ReadCloser, error) {
	return v.impl.VolumeStream(ctx, v.path)
}

func (v *file) Delete(ctx context.Context) (bool, error) {
	return deleteNode(ctx, v.impl, v.path, v.dirmap)
}

const fileDescription = `
This is a file on a remote volume or a container/VM.
`

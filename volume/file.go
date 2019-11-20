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
	rdr    io.ReaderAt
}

// newFile creates a VolumeFile.
func newFile(name string, attr plugin.EntryAttributes, impl Interface, path string) *file {
	vf := &file{
		EntryBase: plugin.NewEntry(name),
	}
	vf.impl = impl
	vf.path = path
	vf.SetAttributes(attr)
	vf.SetTTLOf(plugin.ReadOp, 60*time.Second)

	return vf
}

func (v *file) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(v, "file").SetDescription(fileDescription)
}

func (v *file) Read(ctx context.Context, p []byte, off int64) (int, error) {
	if v.rdr == nil {
		var err error
		if v.rdr, err = v.impl.VolumeRead(ctx, v.path); err != nil {
			return 0, err
		}
	}
	return v.rdr.ReadAt(p, off)
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

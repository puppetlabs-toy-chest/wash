package os

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// ContentCB accepts a path and returns the content associated with that path.
type ContentCB = func(context.Context, string) (plugin.SizedReader, error)

// VolumeFile represents a file in a volume that has content we can access.
type VolumeFile struct {
	plugin.EntryBase
	attr      plugin.Attributes
	contentcb ContentCB
	path      string
}

// NewVolumeFile creates a VolumeFile.
func NewVolumeFile(name string, attr plugin.Attributes, cb ContentCB, path string) *VolumeFile {
	vf := &VolumeFile{
		EntryBase: plugin.NewEntry(name),
		attr:      attr,
		contentcb: cb,
		path:      path,
	}
	vf.CacheConfig().SetTTLOf(plugin.Open, 30*time.Second)

	return vf
}

// Attr returns the attributes of the file.
func (v *VolumeFile) Attr() plugin.Attributes {
	return v.attr
}

// Open returns the content of the file as a SizedReader.
func (v *VolumeFile) Open(ctx context.Context) (plugin.SizedReader, error) {
	return v.contentcb(ctx, v.path)
}

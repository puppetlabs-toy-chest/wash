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
	attr      plugin.Attributes
	contentcb ContentCB
	path      string
}

// NewFile creates a VolumeFile.
func NewFile(name string, attr plugin.Attributes, cb ContentCB, path string) *File {
	vf := &File{
		EntryBase: plugin.NewEntry(name),
		attr:      attr,
		contentcb: cb,
		path:      path,
	}
	vf.CacheConfig().SetTTLOf(plugin.Open, 60*time.Second)

	return vf
}

// Attr returns the attributes of the file.
func (v *File) Attr() plugin.Attributes {
	return v.attr
}

// Open returns the content of the file as a SizedReader.
func (v *File) Open(ctx context.Context) (plugin.SizedReader, error) {
	return v.contentcb(ctx, v.path)
}

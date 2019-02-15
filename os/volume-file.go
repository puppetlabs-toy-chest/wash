package os

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/datastore"
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
	content   datastore.Var
}

// NewVolumeFile creates a VolumeFile.
func NewVolumeFile(name string, attr plugin.Attributes, cb ContentCB, path string) *VolumeFile {
	return &VolumeFile{
		EntryBase: plugin.NewEntry(name),
		attr:      attr,
		contentcb: cb,
		path:      path,
		content:   datastore.NewVar(30 * time.Second),
	}
}

// Attr returns the attributes of the file.
func (v *VolumeFile) Attr() plugin.Attributes {
	return v.attr
}

// Open returns the content of the file as a SizedReader.
func (v *VolumeFile) Open(ctx context.Context) (plugin.SizedReader, error) {
	data, err := v.content.Update(func() (interface{}, error) {
		return v.contentcb(ctx, v.path)
	})
	if err != nil {
		return nil, err
	}
	return data.(plugin.SizedReader), nil
}

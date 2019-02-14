package plugin

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/datastore"
)

// ContentCB accepts a path and returns the content associated with that path.
type ContentCB = func(context.Context, string) (SizedReader, error)

// VolumeFile represents a file in a volume that has content we can access.
type VolumeFile struct {
	EntryBase
	attr      Attributes
	contentcb ContentCB
	path      string
	content   datastore.Var
}

// NewVolumeFile creates a VolumeFile.
func NewVolumeFile(name string, attr Attributes, cb ContentCB, path string) *VolumeFile {
	return &VolumeFile{
		EntryBase: NewEntry(name),
		attr:      attr,
		contentcb: cb,
		path:      path,
		content:   datastore.NewVar(30 * time.Second),
	}
}

// Attr returns the attributes of the file.
func (v *VolumeFile) Attr() Attributes {
	return v.attr
}

// Open returns the content of the file as a SizedReader.
func (v *VolumeFile) Open(ctx context.Context) (SizedReader, error) {
	data, err := v.content.Update(func() (interface{}, error) {
		return v.contentcb(ctx, v.path)
	})
	if err != nil {
		return nil, err
	}
	return data.(SizedReader), nil
}

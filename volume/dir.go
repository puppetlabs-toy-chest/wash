package volume

import (
	"context"
	"io"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// dir represents a directory in a volume. It populates a subtree from the Interface as needed.
type dir struct {
	plugin.EntryBase
	impl Interface
	path string
}

func dirBase() *dir {
	vd := &dir{
		EntryBase: plugin.NewEntryBase(),
	}
	vd.SetLabel("dir")
	return vd
}

// newDir creates a dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, path string) *dir {
	vd := dirBase()
	vd.impl = impl
	vd.path = path
	vd.SetName(name)
	vd.SetAttributes(attr)
	vd.SetTTLOf(plugin.OpenOp, 60*time.Second)
	// Caching handled in List based on 'impl'.
	vd.DisableCachingFor(plugin.ListOp)

	return vd
}

func (v *dir) ChildSchemas() []plugin.EntrySchema {
	return ChildSchemas()
}

// List lists the children of the directory.
func (v *dir) List(ctx context.Context) ([]plugin.Entry, error) {
	return List(ctx, v.impl, v.path)
}

// unDir represents a directory in a volume where we haven't yet explored its contents.
// The new type defers to dir.impl for Interface operations, but acts as a new cache key
// for those operations at the new path.
type unDir struct {
	*dir
}

func (v *unDir) List(ctx context.Context) ([]plugin.Entry, error) {
	// Generate the query using this object as a new cache key.
	return List(ctx, v, v.dir.path)
}

func (v *unDir) VolumeList(ctx context.Context, path string) (DirMap, error) {
	return v.dir.impl.VolumeList(ctx, path)
}

func (v *unDir) VolumeOpen(ctx context.Context, path string) (plugin.SizedReader, error) {
	return v.dir.impl.VolumeOpen(ctx, path)
}

func (v *unDir) VolumeStream(ctx context.Context, path string) (io.ReadCloser, error) {
	return v.dir.impl.VolumeStream(ctx, path)
}

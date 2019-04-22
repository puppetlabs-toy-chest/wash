package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// Dir represents a directory in a volume. It populates a subtree with listcb as needed.
type Dir struct {
	plugin.EntryBase
	impl Interface
	path string
}

// newDir creates a Dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, path string) *Dir {
	vd := &Dir{
		EntryBase: plugin.NewEntry(name),
		impl:      impl,
		path:      path,
	}
	vd.SetAttributes(attr)
	vd.SetTTLOf(plugin.OpenOp, 60*time.Second)
	// Caching handled in List based on 'impl'.
	vd.DisableCachingFor(plugin.ListOp)

	return vd
}

// List lists the children of the directory.
func (v *Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	return List(ctx, v.impl, v.path)
}

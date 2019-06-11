package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// dir represents a directory in a volume. It populates a subtree with listcb as needed.
type dir struct {
	plugin.EntryBase
	impl Interface
	path string
}

func dirTemplate() *dir {
	vd := &dir{
		EntryBase: plugin.NewEntryBase(),
	}
	vd.SetShortType("dir")
	return vd
}

// newDir creates a dir populated from dirs.
func newDir(name string, attr plugin.EntryAttributes, impl Interface, path string) *dir {
	vd := dirTemplate()
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

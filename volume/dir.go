package volume

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/plugin"
)

// ListCB returns a map of volume nodes to their stats, such as that returned by StatParseAll.
type ListCB = func(context.Context) (DirMap, error)

// Dir represents a directory in a volume. It populates a subtree with listcb as needed.
type Dir struct {
	plugin.EntryBase
	cacheEntry plugin.Entry
	listcb     ListCB
	contentcb  ContentCB
	path       string
}

// NewDir creates a Dir populated from dirs.
func NewDir(name string, attr plugin.EntryAttributes, cacheEntry plugin.Entry, lb ListCB, cb ContentCB, path string) *Dir {
	vd := &Dir{
		EntryBase:  plugin.NewEntry(name),
		cacheEntry: cacheEntry,
		listcb:     lb,
		contentcb:  cb,
		path:       path,
	}
	vd.SetAttributes(attr)
	vd.SetTTLOf(plugin.OpenOp, 60*time.Second)
	// Caching handled in MakeEntries on the 'cacheEntry'.
	vd.DisableCachingFor(plugin.ListOp)

	return vd
}

// List lists the children of the directory.
func (v *Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	return MakeEntries(ctx, v.cacheEntry, v.path, v.listcb, v.contentcb)
}

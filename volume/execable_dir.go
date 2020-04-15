package volume

import (
	"context"
	"fmt"

	"github.com/puppetlabs/wash/plugin"
)

type execableInterface interface {
	Interface
	VolumeExec(ctx context.Context, path string, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error)
}

// execableDir adds the exec action to dir.
type execableDir struct {
	dir
	impl execableInterface
}

func newExecDir(name string, attr plugin.EntryAttributes, impl execableInterface, path string) *execableDir {
	e := new(execableDir)
	e.dir = *newDir(name, attr, impl, path)
	e.impl = impl
	return e
}

func (v *execableDir) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	return v.impl.VolumeExec(ctx, v.path, cmd, args, opts)
}

func (v *execableDir) generateChildren(dirmap *dirMap) []plugin.Entry {
	entries := v.dir.generateChildren(dirmap)

	for i, entry := range entries {
		dir, ok := entry.(*dir)
		if ok {
			fmt.Println(dir)
			entries[i] = &execableDir{dir: *dir, impl: v.impl}
		}
	}
	return entries
}

// List lists the children of the directory.
func (v *execableDir) List(ctx context.Context) ([]plugin.Entry, error) {
	if v.dirmap != nil {
		// Children have been pre-populated by a source parent.
		return v.generateChildren(v.dirmap), nil
	}

	// Generate child hierarchy. Don't store it on this entry, but populate new dirs from it.
	dirmap, err := v.impl.VolumeList(ctx, v.path)
	if err != nil {
		return nil, err
	}

	return v.generateChildren(&dirMap{mp: dirmap}), nil
}

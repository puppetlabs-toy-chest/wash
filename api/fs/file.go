package apifs

import (
	"context"
	"os"

	"github.com/puppetlabs/wash/plugin"
)

type file struct {
	*fsnode
}

func newFile(ctx context.Context, finfo os.FileInfo, path string) *file {
	return &file{
		newFSNode(ctx, finfo, path),
	}
}

func (f *file) Read(ctx context.Context, p []byte, off int64) (int, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.ReadAt(p, off)
}

func (f *file) Schema() *plugin.EntrySchema {
	return nil
}

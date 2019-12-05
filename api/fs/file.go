package apifs

import (
	"context"
	"io/ioutil"
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

func (f *file) Read(ctx context.Context) ([]byte, error) {
	content, err := ioutil.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (f *file) Schema() *plugin.EntrySchema {
	return nil
}

var _ = plugin.Readable(&file{})

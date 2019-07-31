package apifs

import (
	"bytes"
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

func (f *file) Open(ctx context.Context) (plugin.SizedReader, error) {
	content, err := ioutil.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(content), nil
}

func (f *file) Schema() *plugin.EntrySchema {
	return nil
}

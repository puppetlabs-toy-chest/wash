package apifs

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/puppetlabs/wash/plugin"
)

type file struct {
	fsnode
}

func newFile(finfo os.FileInfo, path string) *file {
	f := &file{
		newFSNode(finfo, path),
	}
	f.DisableDefaultCaching()
	return f
}

func (f *file) Read(ctx context.Context) ([]byte, error) {
	content, err := ioutil.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (f *file) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(f, "file")
}

var _ = plugin.Readable(&file{})

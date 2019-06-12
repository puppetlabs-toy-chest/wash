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

func fileBase() *file {
	f := &file{
		fsnode: fsnodeBase(),
	}
	f.SetLabel("file")
	return f
}

func newFile(ctx context.Context, finfo os.FileInfo, path string) *file {
	f := fileBase()
	f.build(finfo, path)
	return f
}

func (f *file) Open(ctx context.Context) (plugin.SizedReader, error) {
	content, err := ioutil.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(content), nil
}

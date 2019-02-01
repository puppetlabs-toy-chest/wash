package docker

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

type volume struct {
	*resourcetype
	name string
}

func (cli *volume) Find(ctx context.Context, name string) (plugin.Node, error) {
	return nil, plugin.ENOENT
}

func (cli *volume) List(ctx context.Context) ([]plugin.Node, error) {
	return []plugin.Node{}, nil
}

func (cli *volume) String() string {
	return cli.resourcetype.String() + "/" + cli.Name()
}

func (cli *volume) Name() string {
	return cli.name
}

func (cli *volume) Attr(ctx context.Context) (*plugin.Attributes, error) {
	return &plugin.Attributes{}, nil
}

func (cli *volume) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

package aws

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

type resourcetype interface {
	Find(ctx context.Context, name string) (plugin.Node, error)
	List(ctx context.Context) ([]plugin.Node, error)
	String() string
	Name() string
	Attr(ctx context.Context) (*plugin.Attributes, error)
	Xattr(ctx context.Context) (map[string][]byte, error)
}

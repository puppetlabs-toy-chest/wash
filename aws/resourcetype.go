package aws

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

type resourcetype struct {
	*resources
	typename string
}

func newResourceTypes(resources *resources) map[string]*resourcetype {
	resourcetypes := make(map[string]*resourcetype)
	for _, name := range []string{"s3"} {
		resourcetypes[name] = &resourcetype{resources, name}
	}
	return resourcetypes
}

// Find resource by ID.
func (cli *resourcetype) Find(ctx context.Context, name string) (plugin.Node, error) {
	switch cli.typename {
	case "s3":
		// TODO: Return the specific bucket
		return nil, plugin.ENOENT
	}

	return nil, plugin.ENOTSUP
}

// List all resources.
func (cli *resourcetype) List(ctx context.Context) ([]plugin.Node, error) {
	switch cli.typename {
	case "s3":
		// TODO: Return all of the buckets
		return []plugin.Node{}, nil
	}

	return nil, plugin.ENOTSUP
}

// A unique string describing the resource type.
func (cli *resourcetype) String() string {
	return cli.resources.String() + "/" + cli.Name()
}

// Name returns the name of the resource type.
func (cli *resourcetype) Name() string {
	return cli.typename
}

// Attr returns attributes of the resource type.
func (cli *resourcetype) Attr(ctx context.Context) (*plugin.Attributes, error) {
	latest := cli.updated
	switch cli.typename {
	case "s3":
		// TODO: Compare with the mtimes of the buckets and return whatever's the latest
		return plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
	}

	return nil, plugin.ENOENT
}

// Xattr returns a map of extended attributes.
func (cli *resourcetype) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

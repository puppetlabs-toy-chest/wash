package aws

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

type s3 struct {
	*resources
}

func newS3ResourceType(resources *resources) *s3 {
	return &s3{resources}
}

// Find the bucket by ID
func (cli *s3) Find(ctx context.Context, name string) (plugin.Node, error) {
	// TODO: Return the specific bucket
	return nil, plugin.ENOENT
}

// List all buckets
func (cli *s3) List(ctx context.Context) ([]plugin.Node, error) {
	// TODO: Return all of the buckets
	return []plugin.Node{}, nil
}

// A unique string describing the S3 resource type.
func (cli *s3) String() string {
	return cli.resources.String() + "/" + cli.Name()
}

// Name returns the name of the S3 resource type.
func (cli *s3) Name() string {
	return "s3"
}

// Attr returns attributes of the S3 resource type.
func (cli *s3) Attr(ctx context.Context) (*plugin.Attributes, error) {
	latest := cli.updated

	// TODO: Compare with the mtimes of the buckets and return whatever's the latest
	return &plugin.Attributes{Mtime: latest}, nil
}

// Xattr returns a map of extended attributes.
func (cli *s3) Xattr(ctx context.Context) (map[string][]byte, error) {
	// TODO: What to return here?
	return map[string][]byte{}, nil
}

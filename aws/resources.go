package aws

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type resources struct {
	*root
	updated       time.Time
	resourcetypes map[string]resourcetype
}

func newResources(cli *root) *resources {
	resources := &resources{root: cli, updated: time.Now()}

	resources.resourcetypes = make(map[string]resourcetype)
	resources.resourcetypes["s3"] = newS3ResourceType(resources)

	return resources
}

// Find the resource type by its ID.
func (cli *resources) Find(ctx context.Context, name string) (plugin.Node, error) {
	if rt, ok := cli.resourcetypes[name]; ok {
		log.Debugf("Found resource type %v", rt)
		return plugin.NewDir(rt), nil
	}

	return nil, plugin.ENOENT
}

// List the available resource types as directories
func (cli *resources) List(ctx context.Context) ([]plugin.Node, error) {
	log.Debugf("Listing %v resource types in %v", len(cli.resourcetypes), cli)
	entries := make([]plugin.Node, 0, len(cli.resourcetypes))
	for _, rt := range cli.resourcetypes {
		entries = append(entries, plugin.NewDir(rt))
	}
	return entries, nil
}

func (cli *resources) String() string {
	return cli.root.Name() + "/" + cli.Name()
}

func (cli *resources) Name() string {
	return "resources"
}

// Attr returns attributes of the named resource.
func (cli *resources) Attr(ctx context.Context) (*plugin.Attributes, error) {
	// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
	latest := cli.updated
	for _, v := range cli.resourcetypes {
		attr, err := v.Attr(ctx)
		if err != nil {
			return nil, err
		}
		if attr.Mtime.After(latest) {
			latest = attr.Mtime
		}
	}
	return &plugin.Attributes{Mtime: latest}, nil
}

// Xattr returns a map of extended attributes.
func (cli *resources) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

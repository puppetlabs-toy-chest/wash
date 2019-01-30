package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type resourcetype struct {
	*namespace
	typename string
}

func newResourceTypes(ns *namespace) map[string]*resourcetype {
	resourcetypes := make(map[string]*resourcetype)
	for _, name := range []string{"pod"} {
		resourcetypes[name] = &resourcetype{ns, name}
	}
	return resourcetypes
}

// Find resource by ID.
func (cli *resourcetype) Find(ctx context.Context, name string) (plugin.Node, error) {
	switch cli.typename {
	case "pod":
		if pods, err := cli.cachedPods(ctx, cli.name); err == nil {
			if id, ok := datastore.FindCompositeString(pods, name); ok {
				log.Debugf("Found pod %v in /kubernetes/%v", id, cli.name)
				return plugin.NewFile(newPod(cli.client, id)), nil
			}
		}
		log.Debugf("Did not find %v in /kubernetes/%v", name, cli.name)
		return nil, plugin.ENOENT
	}
	return nil, plugin.ENOTSUP
}

// List all resources as files.
func (cli *resourcetype) List(ctx context.Context) ([]plugin.Node, error) {
	switch cli.typename {
	case "pod":
		pods, err := cli.cachedPods(ctx, cli.name)
		if err != nil {
			return nil, err
		}
		log.Debugf("Listing %v pods in /kubernetes/%v", len(pods), cli.name)
		entries := make([]plugin.Node, len(pods))
		for i, id := range pods {
			entries[i] = plugin.NewFile(newPod(cli.client, id))
		}
		return entries, nil
	}
	return nil, plugin.ENOTSUP
}

// Name returns the name of the resource type.
func (cli *resourcetype) Name() string {
	return cli.typename
}

// Attr returns attributes of the resource type.
func (cli *resourcetype) Attr(ctx context.Context) (*plugin.Attributes, error) {
	// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
	// TODO: make this more constrained to the specific resource.
	latest := cli.updated
	for _, v := range cli.reqs {
		if updated := v.LastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *resourcetype) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

package kubernetes

import (
	"context"
	"sync"

	"github.com/puppetlabs/wash/datastore"
	log "github.com/sirupsen/logrus"
	"github.com/puppetlabs/wash/plugin"
)

type resourcetype struct {
	*namespace
	typename string
	reqs     sync.Map
}

func newResourceTypes(ns *namespace) map[string]*resourcetype {
	resourcetypes := make(map[string]*resourcetype)
	// Use individual caches for slower resources like volumes to control the timeout.
	for _, name := range []string{"pod", "pvc"} {
		resourcetypes[name] = &resourcetype{ns, name, sync.Map{}}
	}
	return resourcetypes
}

// Find resource by ID.
func (cli *resourcetype) Find(ctx context.Context, name string) (plugin.Node, error) {
	switch cli.typename {
	case "pod":
		if pods, err := cli.cachedPods(ctx, cli.name); err == nil {
			if id, ok := datastore.FindCompositeString(pods, name); ok {
				log.Debugf("Found pod %v in %v", id, cli)
				return plugin.NewFile(newPod(cli, id)), nil
			}
		}
		log.Debugf("Did not find %v in %v", name, cli)
		return nil, plugin.ENOENT
	case "pvc":
		if pvcs, err := cli.cachedPvcs(ctx, cli.name); err == nil {
			if id, ok := datastore.FindCompositeString(pvcs, name); ok {
				log.Debugf("Found persistent volume claim %v in %v", id, cli)
				return plugin.NewDir(newPvc(cli, id)), nil
			}
		}
		log.Debugf("Did not find %v in %v", name, cli)
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
		log.Debugf("Listing %v pods in %v", len(pods), cli)
		entries := make([]plugin.Node, len(pods))
		for i, id := range pods {
			entries[i] = plugin.NewFile(newPod(cli, id))
		}
		return entries, nil
	case "pvc":
		pvcs, err := cli.cachedPvcs(ctx, cli.name)
		if err != nil {
			return nil, err
		}
		log.Debugf("Listing %v pvcs in %v", len(pvcs), cli)
		entries := make([]plugin.Node, len(pvcs))
		for i, id := range pvcs {
			entries[i] = plugin.NewDir(newPvc(cli, id))
		}
		return entries, nil
	}
	return nil, plugin.ENOTSUP
}

// A unique string describing the resource type.
func (cli *resourcetype) String() string {
	return cli.namespace.String() + "/" + cli.Name()
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
	cli.reqs.Range(func(k, v interface{}) bool {
		if updated := v.(*datastore.StreamBuffer).LastUpdate(); updated.After(latest) {
			latest = updated
		}
		return true
	})
	return &plugin.Attributes{Mtime: latest}, nil
}

// Xattr returns a map of extended attributes.
func (cli *resourcetype) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

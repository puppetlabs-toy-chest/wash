package kubernetes

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/puppetlabs/wash/plugin"
)

type namespace struct {
	*client
	name          string
	updated       time.Time
	resourcetypes map[string]*resourcetype
}

func newNamespace(cli *client, name string) *namespace {
	ns := &namespace{cli, name, time.Now(), nil}
	ns.resourcetypes = newResourceTypes(ns)
	return ns
}

// Find resource type by name.
func (cli *namespace) Find(ctx context.Context, name string) (plugin.Node, error) {
	if rt, ok := cli.resourcetypes[name]; ok {
		log.Debugf("Found resource type %v", rt)
		return plugin.NewDir(rt), nil
	}
	return nil, plugin.ENOENT
}

// List all running pods as files.
func (cli *namespace) List(ctx context.Context) ([]plugin.Node, error) {
	log.Debugf("Listing %v resource types in %v", len(cli.resourcetypes), cli)
	entries := make([]plugin.Node, 0, len(cli.resourcetypes))
	for _, rt := range cli.resourcetypes {
		entries = append(entries, plugin.NewDir(rt))
	}
	return entries, nil
}

// A unique string describing the namespace.
func (cli *namespace) String() string {
	return cli.client.Name() + "/" + cli.Name()
}

// Name returns the namespace name.
func (cli *namespace) Name() string {
	return cli.name
}

// Attr returns attributes of the namespace.
func (cli *namespace) Attr(ctx context.Context) (*plugin.Attributes, error) {
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
func (cli *namespace) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

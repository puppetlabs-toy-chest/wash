package kubernetes

import (
	"context"
	"sort"

	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

type node struct {
	*client
	name   string
	parent *node
}

// Find container by ID.
func (cli *node) Find(ctx context.Context, name string) (plugin.Node, error) {
	if cli.parent == nil {
		switch cli.name {
		case "pods":
			if pd, err := cli.cachedPodFind(ctx, name); err == nil {
				log.Debugf("Found pod %v, %v", name, pd)
				return plugin.NewFile(&pod{cli.client, name}), nil
			}
		case "namespaces":
			if namespace, err := cli.cachedNamespaceFind(ctx, name); err == nil {
				log.Debugf("Found namespace %v, %v", name, namespace)
				return plugin.NewDir(&node{cli.client, name, cli}), nil
			}
		}
		return nil, plugin.ENOENT
	}

	if cli.parent.name == "namespaces" {
		if pods, err := cli.cachedNamespaceFind(ctx, cli.name); err == nil {
			idx := sort.SearchStrings(pods, name)
			if pods[idx] == name {
				log.Debugf("Found pod %v in namespace %v", name, cli.name)
				return plugin.NewFile(&pod{cli.client, name}), nil
			}
		}
	}
	return nil, plugin.ENOENT
}

// List all running pods as files.
func (cli *node) List(ctx context.Context) ([]plugin.Node, error) {
	if cli.parent == nil {
		switch cli.name {
		case "pods":
			pods, err := cli.cachedPodList(ctx)
			if err != nil {
				return nil, err
			}
			log.Debugf("Listing %v pods in /kubernetes/pods", len(pods))
			entries := make([]plugin.Node, len(pods))
			for i, v := range pods {
				entries[i] = plugin.NewFile(&pod{cli.client, v})
			}
			return entries, nil
		case "namespaces":
			namespaces, err := cli.cachedNamespaceList(ctx)
			if err != nil {
				return nil, err
			}
			log.Debugf("Listing %v namespaces in /kubernetes/namespaces", len(namespaces))
			entries := make([]plugin.Node, len(namespaces))
			for i, v := range namespaces {
				entries[i] = plugin.NewDir(&node{cli.client, v, cli})
			}
			return entries, nil
		}
		return []plugin.Node{}, nil
	}

	if cli.parent.name == "namespaces" {
		pods, err := cli.cachedNamespaceFind(ctx, cli.name)
		if err != nil {
			return nil, err
		}
		log.Debugf("Listing %v pods in /kubernetes/namespaces/%v", len(pods), cli.name)
		entries := make([]plugin.Node, len(pods))
		for i, v := range pods {
			entries[i] = plugin.NewFile(&pod{cli.client, v})
		}
		return entries, nil
	}
	return []plugin.Node{}, nil
}

// Name returns the node's name.
func (cli *node) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *node) Attr(ctx context.Context) (*plugin.Attributes, error) {
	// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
	// TODO: make this more constrained for namespaces.
	latest := cli.updated
	for _, v := range cli.reqs {
		if updated := v.LastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *node) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

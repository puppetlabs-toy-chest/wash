package kubernetes

import (
	"context"
	
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type namespace struct {
	plugin.EntryBase
	client        *k8s.Clientset
	config        *rest.Config
	resources     []plugin.Entry
}

func namespaceTemplate() *namespace {
	ns := &namespace{
		EntryBase: plugin.NewEntry(),
	}
	ns.SetShortType("namespace")
	return ns
}

func newNamespace(name string, meta *corev1.Namespace, c *k8s.Clientset, cfg *rest.Config) *namespace {
	ns := namespaceTemplate()
	ns.client = c
	ns.config = cfg
	ns.SetName(name)
	ns.resources = []plugin.Entry{
		newPodsDir(ns),
		newPVCSDir(ns),
	}
	// TODO: Figure out other attributes that we could set here, if any.
	ns.Attributes().SetMeta(meta)
	return ns
}

func (n *namespace) ChildSchemas() []plugin.EntrySchema {
	return plugin.ChildSchemas(podsDirTemplate(), pvcsDirTemplate())
}

func (n *namespace) List(ctx context.Context) ([]plugin.Entry, error) {
	return n.resources, nil
}
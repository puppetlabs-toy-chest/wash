package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type namespace struct {
	plugin.EntryBase
	client    *k8s.Clientset
	config    *rest.Config
	resources []plugin.Entry
}

func newNamespace(name string, meta *corev1.Namespace, c *k8s.Clientset, cfg *rest.Config) *namespace {
	ns := &namespace{
		EntryBase: plugin.NewEntry(name),
	}
	ns.client = c
	ns.config = cfg
	ns.resources = []plugin.Entry{
		newPodsDir(ns),
		newPVCSDir(ns),
	}
	// TODO: Figure out other attributes that we could set here, if any.
	ns.Attributes().SetMeta(meta)
	return ns
}

func (n *namespace) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(n, "namespace").
		SetMetaAttributeSchema(corev1.Namespace{}).
		SetDescription(namespaceDescription)
}

func (n *namespace) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&podsDir{}).Schema(),
		(&pvcsDir{}).Schema(),
	}
}

func (n *namespace) List(ctx context.Context) ([]plugin.Entry, error) {
	return n.resources, nil
}

func (n *namespace) Delete(ctx context.Context) (bool, error) {
	err := n.client.CoreV1().Namespaces().Delete(n.Name(), &v1.DeleteOptions{})
	return true, err
}

const namespaceDescription = `
This is a Kubernetes namespace.
`

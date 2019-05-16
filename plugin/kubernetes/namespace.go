package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type namespace struct {
	plugin.EntryBase
	client        *k8s.Clientset
	config        *rest.Config
	resourcetypes []plugin.Entry
}

func newNamespace(name string, meta *corev1.Namespace, c *k8s.Clientset, cfg *rest.Config) *namespace {
	ns := &namespace{EntryBase: plugin.NewEntry(name), client: c, config: cfg}
	ns.resourcetypes = []plugin.Entry{
		&pods{plugin.NewEntry("pods"), c, cfg, name},
		&pvcs{plugin.NewEntry("persistentvolumeclaims"), c, name},
	}
	// TODO: Figure out other attributes that we could set here, if any.
	attr := plugin.EntryAttributes{}
	attr.SetMeta(meta)
	ns.SetAttributes(attr)
	return ns
}

func (n *namespace) List(ctx context.Context) ([]plugin.Entry, error) {
	activity.Record(ctx, "Listing resource types for namespace %v: %v", n.Name(), n.resourcetypes)
	return n.resourcetypes, nil
}
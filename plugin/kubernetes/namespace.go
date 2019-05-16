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
	metadata      *corev1.Namespace
	client        *k8s.Clientset
	config        *rest.Config
	resourcetypes []plugin.Entry
}

func newNamespace(name string, meta *corev1.Namespace, c *k8s.Clientset, cfg *rest.Config) *namespace {
	ns := &namespace{EntryBase: plugin.NewEntry(name), metadata: meta, client: c, config: cfg}
	ns.resourcetypes = []plugin.Entry{
		&pods{plugin.NewEntry("pods"), c, cfg, name},
		&pvcs{plugin.NewEntry("persistentvolumeclaims"), c, name},
	}
	return ns
}

func (n *namespace) List(ctx context.Context) ([]plugin.Entry, error) {
	activity.Record(ctx, "Listing resource types for namespace %v: %v", n.Name(), n.resourcetypes)
	return n.resourcetypes, nil
}

func (n *namespace) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	activity.Record(ctx, "Metadata for namespace %v: %+v", n.Name(), n.metadata)
	if n.metadata != nil {
		return plugin.ToJSONObject(n.metadata), nil
	}
	return plugin.JSONObject{}, nil
}

package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type k8context struct {
	plugin.EntryBase
	client    *k8s.Clientset
	config    *rest.Config
	defaultns string
}

func newK8Context(name string, client *k8s.Clientset, config *rest.Config, defaultns string) *k8context {
	context := &k8context{
		EntryBase: plugin.NewEntry(name),
	}
	context.client = client
	context.config = config
	context.defaultns = defaultns
	return context
}

func (c *k8context) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(c, "context")
}

func (c *k8context) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&namespace{}).Schema(),
	}
}

func (c *k8context) List(ctx context.Context) ([]plugin.Entry, error) {
	nsi := c.client.CoreV1().Namespaces()
	nsList, err := nsi.List(metav1.ListOptions{})
	if err != nil {
		activity.Record(ctx, "Error loading namespaces, using default namespace %v: %v", c.defaultns, err)
		ns, err := nsi.Get(c.defaultns, metav1.GetOptions{})
		if err != nil {
			activity.Record(ctx, "Error loading default namespace, metadata will not be available: %v", err)
		}
		return []plugin.Entry{newNamespace(c.defaultns, ns, c.client, c.config)}, nil
	}

	namespaces := make([]plugin.Entry, len(nsList.Items))
	for i, ns := range nsList.Items {
		namespaces[i] = newNamespace(ns.Name, &ns, c.client, c.config)
	}
	activity.Record(ctx, "Listing namespaces: %+v", namespaces)
	return namespaces, nil
}

package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/journal"
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

func (c *k8context) List(ctx context.Context) ([]plugin.Entry, error) {
	nsi := c.client.CoreV1().Namespaces()
	nsList, err := nsi.List(metav1.ListOptions{})
	if err != nil {
		journal.Record(ctx, "Error loading namespaces, using default namespace %v: %v", c.defaultns, err)
		ns, err := nsi.Get(c.defaultns, metav1.GetOptions{})
		if err != nil {
			journal.Record(ctx, "Error loading default namespace, metadata will not be available: %v", err)
		}
		return []plugin.Entry{newNamespace(c, c.defaultns, ns, c.client, c.config)}, nil
	}

	namespaces := make([]plugin.Entry, len(nsList.Items))
	for i, ns := range nsList.Items {
		namespaces[i] = newNamespace(c, ns.Name, &ns, c.client, c.config)
	}
	journal.Record(ctx, "Listing namespaces: %+v", namespaces)
	return namespaces, nil
}

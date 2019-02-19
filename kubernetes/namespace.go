package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
)

type namespace struct {
	plugin.EntryBase
	metadata      *corev1.Namespace
	client        *k8s.Clientset
	resourcetypes []plugin.Entry
}

func newNamespace(name string, meta *corev1.Namespace, c *k8s.Clientset) *namespace {
	ns := &namespace{EntryBase: plugin.NewEntry(name), metadata: meta, client: c}
	ns.resourcetypes = []plugin.Entry{
		&pods{plugin.NewEntry("pods"), c, name},
		&pvcs{plugin.NewEntry("persistentvolumeclaims"), c, name},
	}
	return ns
}

func (n *namespace) LS(ctx context.Context) ([]plugin.Entry, error) {
	log.Debugf("Listing %v resource types in %v", len(n.resourcetypes), n)
	return n.resourcetypes, nil
}

func (n *namespace) Metadata(ctx context.Context) (plugin.MetadataMap, error) {
	if n.metadata != nil {
		return plugin.ToMetadata(n.metadata), nil
	}
	return plugin.MetadataMap{}, nil
}

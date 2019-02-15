package kubernetes

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/puppetlabs/wash/plugin"
	k8s "k8s.io/client-go/kubernetes"
)

type namespace struct {
	plugin.EntryBase
	client        *k8s.Clientset
	resourcetypes []plugin.Entry
}

func newNamespace(name string, c *k8s.Clientset) *namespace {
	ns := &namespace{EntryBase: plugin.NewEntry(name), client: c}
	ns.resourcetypes = []plugin.Entry{
		&pods{plugin.NewEntry("pods"), c, name},
	}
	return ns
}

func (n *namespace) LS(ctx context.Context) ([]plugin.Entry, error) {
	log.Debugf("Listing %v resource types in %v", len(n.resourcetypes), n)
	return n.resourcetypes, nil
}

// TODO: implement Metadata

package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

type k8context struct {
	plugin.EntryBase
	client    *k8s.Clientset
	defaultns string
}

func (c *k8context) LS(ctx context.Context) ([]plugin.Entry, error) {
	nsList, err := c.client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Printf("Error loading namespaces, using default namespace %v: %v", c.defaultns, err)
		return []plugin.Entry{newNamespace(c.defaultns, c.client)}, nil
	}

	namespaces := make([]plugin.Entry, len(nsList.Items))
	for i, ns := range nsList.Items {
		namespaces[i] = newNamespace(ns.Name, c.client)
	}
	return namespaces, nil
}

// TODO: implement Metadata

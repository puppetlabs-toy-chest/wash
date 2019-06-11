package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type podsDir struct {
	plugin.EntryBase
	client *k8s.Clientset
	config *rest.Config
	ns     string
}

func podsDirTemplate() *podsDir {
	pds := &podsDir{
		EntryBase: plugin.NewEntryBase(),
	}
	pds.SetName("pods").IsSingleton()
	return pds
}

func newPodsDir(ns *namespace) *podsDir {
	pds := podsDirTemplate()
	pds.client = ns.client
	pds.config = ns.config
	pds.ns = ns.Name()
	return pds
}

func (ps *podsDir) ChildSchemas() []plugin.EntrySchema {
	return plugin.ChildSchemas(podTemplate())
}

func (ps *podsDir) List(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: identify whether we have permission to get logs for this namespace early, so
	// we can return quickly for Attributes.
	podList, err := ps.client.CoreV1().Pods(ps.ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	entries := make([]plugin.Entry, len(podList.Items))
	for i, p := range podList.Items {
		pd, err := newPod(ctx, ps.client, ps.config, ps.ns, &p)
		if err != nil {
			return nil, err
		}

		entries[i] = pd
	}
	return entries, nil
}

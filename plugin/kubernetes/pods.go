package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type pods struct {
	plugin.EntryBase
	client *k8s.Clientset
	config *rest.Config
	ns     string
}

func (ps *pods) List(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: identify whether we have permission to get logs for this namespace early, so
	// we can return quickly for Attributes.
	podList, err := ps.client.CoreV1().Pods(ps.ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	entries := make([]plugin.Entry, len(podList.Items))
	for i, p := range podList.Items {
		entries[i] = newPod(ps.client, ps.config, ps.ns, &p)
	}
	return entries, nil
}

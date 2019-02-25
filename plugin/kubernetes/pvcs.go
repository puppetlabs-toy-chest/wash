package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

type pvcs struct {
	plugin.EntryBase
	client *k8s.Clientset
	ns     string
}

func (pv *pvcs) LS(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: identify whether we have permission to run pods for this namespace early, so
	// we can return quickly on expensive commands.
	pvcI := pv.client.CoreV1().PersistentVolumeClaims(pv.ns)
	pvcList, err := pvcI.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	entries := make([]plugin.Entry, len(pvcList.Items))
	for i, p := range pvcList.Items {
		entries[i] = newPVC(pvcI, pv.client.CoreV1().Pods(pv.ns), &p)
	}
	return entries, nil
}

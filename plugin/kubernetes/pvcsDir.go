package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type pvcsDir struct {
	plugin.EntryBase
	client *k8s.Clientset
	config *rest.Config
	ns     string
}

func newPVCSDir(ns *namespace) *pvcsDir {
	pv := &pvcsDir{
		EntryBase: plugin.NewEntry("persistentvolumeclaims"),
	}
	pv.client = ns.client
	pv.config = ns.config
	pv.ns = ns.Name()
	return pv
}

func (pv *pvcsDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(pv, "persistentvolumeclaims").IsSingleton()
}

func (pv *pvcsDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&pvc{}).Schema(),
	}
}

func (pv *pvcsDir) List(ctx context.Context) ([]plugin.Entry, error) {
	// TODO: identify whether we have permission to run pods for this namespace early, so
	// we can return quickly on expensive commands.
	pvcI := pv.client.CoreV1().PersistentVolumeClaims(pv.ns)
	pvcList, err := pvcI.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	entries := make([]plugin.Entry, len(pvcList.Items))
	for i, p := range pvcList.Items {
		entries[i] = newPVC(pvcI, pv.client, pv.config, pv.ns, &p)
	}
	return entries, nil
}

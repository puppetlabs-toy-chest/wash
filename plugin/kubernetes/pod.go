package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type pod struct {
	plugin.EntryBase
	client *k8s.Clientset
	config *rest.Config
	ns     string
}

func newPod(ctx context.Context, client *k8s.Clientset, config *rest.Config, ns string, p *corev1.Pod) (*pod, error) {
	pd := &pod{
		EntryBase: plugin.NewEntry(p.Name),
	}
	pd.client = client
	pd.config = config
	pd.ns = ns

	pd.
		SetPartialMetadata(p).
		Attributes().
		SetCrtime(p.CreationTimestamp.Time).
		SetAtime(p.CreationTimestamp.Time)

	return pd, nil
}

func (p *pod) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(p, "pod").
		SetPartialMetadataSchema(corev1.Pod{})
}

func (p *pod) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&container{}).Schema(),
	}
}

func (p *pod) List(ctx context.Context) ([]plugin.Entry, error) {
	pd, err := p.client.CoreV1().Pods(p.ns).Get(p.Name(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	entries := make([]plugin.Entry, len(pd.Spec.Containers))
	for i, c := range pd.Spec.Containers {
		c, err := newContainer(ctx, p.client, p.config, p.ns, &c, pd)
		if err != nil {
			return nil, err
		}

		entries[i] = c
	}

	return entries, nil
}

func (p *pod) Delete(ctx context.Context) (bool, error) {
	err := p.client.CoreV1().Pods(p.ns).Delete(p.Name(), &metav1.DeleteOptions{})
	return true, err
}

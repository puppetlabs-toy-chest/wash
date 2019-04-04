// Package kubernetes presents a filesystem hierarchy for Kubernetes resources.
//
// It uses uses contexts from ~/.kube/config to access Kubernetes APIs.
package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	// Loads the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Root of the Kubernetes plugin
type Root struct {
	plugin.EntryBase
	contexts []plugin.Entry
}

func (r *Root) createContext(raw clientcmdapi.Config, name string, access clientcmd.ConfigAccess) (plugin.Entry, error) {
	config := clientcmd.NewNonInteractiveClientConfig(raw, name, &clientcmd.ConfigOverrides{}, access)
	cfg, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := k8s.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	defaultns, _, err := config.Namespace()
	if err != nil {
		return nil, err
	}
	return &k8context{r.NewEntry(name), clientset, cfg, defaultns}, nil
}

// Init for root
func (r *Root) Init() error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	raw, err := config.RawConfig()
	if err != nil {
		return err
	}

	r.EntryBase = plugin.NewRootEntry("kubernetes")
	r.DisableDefaultCaching()

	contexts := make([]plugin.Entry, 0)
	for name := range raw.Contexts {
		ctx, err := r.createContext(raw, name, config.ConfigAccess())
		if err != nil {
			journal.Record(context.Background(), "loading context %v failed: %+v", name, err)
			continue
		}
		contexts = append(contexts, ctx)
	}
	r.contexts = contexts

	return nil
}

// List returns available contexts.
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	return r.contexts, nil
}

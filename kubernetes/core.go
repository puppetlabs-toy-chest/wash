package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Loads the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Root of the Kubernetes plugin
type Root struct {
	contexts []plugin.Entry
}

// Init for root
func (r *Root) Init() error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	raw, err := config.RawConfig()
	if err != nil {
		return err
	}

	contexts := make([]plugin.Entry, 0)
	for name := range raw.Contexts {
		config = clientcmd.NewNonInteractiveClientConfig(raw, name, configOverrides, config.ConfigAccess())
		cfg, err := config.ClientConfig()
		if err != nil {
			return err
		}
		clientset, err := k8s.NewForConfig(cfg)
		if err != nil {
			return err
		}
		defaultns, _, err := config.Namespace()
		if err != nil {
			return err
		}
		contexts = append(contexts, &k8context{plugin.NewEntry(name), clientset, defaultns})
	}
	r.contexts = contexts
	return nil
}

// Name returns 'docker'
func (r *Root) Name() string {
	return "kubernetes"
}

// LS returns available contexts.
func (r *Root) LS(ctx context.Context) ([]plugin.Entry, error) {
	return r.contexts, nil
}

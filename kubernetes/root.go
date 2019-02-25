package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
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

func createContext(raw clientcmdapi.Config, name string, access clientcmd.ConfigAccess) (plugin.Entry, error) {
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
	return &k8context{plugin.NewEntry(name), clientset, cfg, defaultns}, nil
}

// Init for root
func (r *Root) Init() error {
	r.EntryBase = plugin.NewEntry("kubernetes")
	r.CacheConfig().TurnOffCaching()

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	raw, err := config.RawConfig()
	if err != nil {
		return err
	}

	contexts := make([]plugin.Entry, 0)
	for name := range raw.Contexts {
		ctx, err := createContext(raw, name, config.ConfigAccess())
		if err != nil {
			log.Warnf("loading context %v failed: %+v", name, err)
			continue
		}
		contexts = append(contexts, ctx)
	}
	r.contexts = contexts

	return nil
}

// LS returns available contexts.
func (r *Root) LS(ctx context.Context) ([]plugin.Entry, error) {
	return r.contexts, nil
}

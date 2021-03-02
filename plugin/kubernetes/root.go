// Package kubernetes presents a filesystem hierarchy for Kubernetes resources.
//
// It uses uses contexts from ~/.kube/config to access Kubernetes APIs.
package kubernetes

import (
	"context"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	// Loads all available auth plugins (required to authenticate against GKE, Azure, OpenStack and OIDC clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Root of the Kubernetes plugin
type Root struct {
	plugin.EntryBase
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
	return newK8Context(name, clientset, cfg, defaultns), nil
}

// Init for root
func (r *Root) Init(map[string]interface{}) error {
	r.EntryBase = plugin.NewEntry("kubernetes")
	r.DisableDefaultCaching()

	return nil
}

// Schema returns the root's schema
func (r *Root) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(r, "kubernetes").
		SetDescription(rootDescription).
		IsSingleton()
}

// ChildSchemas returns the root's child schemas
func (r *Root) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&k8context{}).Schema(),
	}
}

// WrappedTypes implements plugin.Root#WrappedTypes
func (r *Root) WrappedTypes() plugin.SchemaMap {
	return map[interface{}]*plugin.JSONSchema{
		v1.Time{}:           plugin.TimeSchema(),
		resource.Quantity{}: plugin.StringSchema(),
	}
}

// List returns available contexts.
func (r *Root) List(ctx context.Context) ([]plugin.Entry, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	raw, err := config.RawConfig()
	if err != nil {
		return nil, err
	}

	contexts := make([]plugin.Entry, 0)
	for name := range raw.Contexts {
		ctx, err := createContext(raw, name, config.ConfigAccess())
		if err != nil {
			activity.Warnf(context.Background(), "loading context %v failed: %+v", name, err)
			continue
		}
		contexts = append(contexts, ctx)
	}

	return contexts, nil
}

const rootDescription = `
This is the Kubernetes plugin root. It lets you interact with Kubernetes resources
like pods and persistent volume claims.

Kubernetes contexts are extracted from ~/.kube/config.
`

package kubernetes

import (
	"context"
	"sync"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Loads the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type client struct {
	*k8s.Clientset
	cache      *datastore.MemCache
	nsmux      sync.RWMutex
	namespaces map[string]*namespace
	updated    time.Time
	root       string
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond
const allNamespace = "all"

// ListContexts lists the available kubernetes contexts.
func ListContexts() (map[string]clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	raw, err := config.RawConfig()
	if err != nil {
		return nil, err
	}

	configs := make(map[string]clientcmd.ClientConfig)
	for name := range raw.Contexts {
		configs[name] = clientcmd.NewNonInteractiveClientConfig(raw, name, configOverrides, config.ConfigAccess())
	}
	return configs, nil
}

// Create a new kubernetes client.
func Create(name string, context interface{}, cache *datastore.MemCache) (plugin.DirProtocol, error) {
	config, err := context.(clientcmd.ClientConfig).ClientConfig()
	if err != nil {
		return nil, err
	}
	// create the clientset
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	namespaces := make(map[string]*namespace)
	return &client{clientset, cache, sync.RWMutex{}, namespaces, time.Now(), name}, nil
}

// Find namespace.
func (cli *client) Find(ctx context.Context, name string) (plugin.Node, error) {
	cli.refreshNamespaces(ctx)
	cli.nsmux.RLock()
	defer cli.nsmux.RUnlock()
	if ns, ok := cli.namespaces[name]; ok {
		return plugin.NewDir(ns), nil
	}
	return nil, plugin.ENOENT
}

// List all namespaces.
func (cli *client) List(ctx context.Context) ([]plugin.Node, error) {
	cli.refreshNamespaces(ctx)
	cli.nsmux.RLock()
	defer cli.nsmux.RUnlock()
	log.Debugf("Listing %v namespaces in %v", len(cli.namespaces), cli.Name())
	entries := make([]plugin.Node, 0, len(cli.namespaces))
	for _, ns := range cli.namespaces {
		entries = append(entries, plugin.NewDir(ns))
	}
	return entries, nil
}

// Name returns the root directory of the client.
func (cli *client) Name() string {
	return cli.root
}

// Attr returns attributes of the named resource.
func (cli *client) Attr(ctx context.Context) (*plugin.Attributes, error) {
	// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
	latest := cli.updated
	cli.nsmux.RLock()
	defer cli.nsmux.RUnlock()
	for _, v := range cli.namespaces {
		attr, err := v.Attr(ctx)
		if err != nil {
			return nil, err
		}
		if attr.Mtime.After(latest) {
			latest = attr.Mtime
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *client) Xattr(ctx context.Context) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

func (cli *client) refreshNamespaces(ctx context.Context) error {
	cli.nsmux.Lock()
	defer cli.nsmux.Unlock()
	namespaces, err := cli.cachedNamespaces(ctx)
	if err != nil {
		return err
	}

	// Remove unnamed namespaces
	for name := range cli.namespaces {
		if name == allNamespace {
			// Don't remove 'all' namespace.
			continue
		}
		if !datastore.ContainsString(namespaces, name) {
			delete(cli.namespaces, name)
		}
	}

	// Ensure 'all' namespace is always included.
	for _, name := range append(namespaces, allNamespace) {
		if _, ok := cli.namespaces[name]; ok {
			continue
		}

		cli.namespaces[name] = newNamespace(cli, name)
	}
	return nil
}

func (cli *client) cachedNamespaces(ctx context.Context) ([]string, error) {
	return datastore.CachedStrings(cli.cache, cli.Name(), func() ([]string, error) {
		nsList, err := cli.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		namespaces := make([]string, len(nsList.Items))
		for i, ns := range nsList.Items {
			namespaces[i] = ns.Name
		}
		return namespaces, nil
	})
}

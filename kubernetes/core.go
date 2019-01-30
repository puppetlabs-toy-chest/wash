package kubernetes

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Loads the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type client struct {
	*k8s.Clientset
	cache      *bigcache.BigCache
	nsmux      sync.Mutex
	namespaces map[string]*namespace
	mux        sync.Mutex
	reqs       map[string]*datastore.StreamBuffer
	updated    time.Time
	root       string
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond
const allNamespace = "all"

// Create a new kubernetes client.
func Create(name string) (plugin.DirProtocol, error) {
	var kubeconfig *string
	if h := os.Getenv("HOME"); h != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(h, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// TODO: this should be a helper, or passed to Create.
	cacheconfig := bigcache.DefaultConfig(5 * time.Second)
	cacheconfig.CleanWindow = 100 * time.Millisecond
	cache, err := bigcache.NewBigCache(cacheconfig)
	if err != nil {
		return nil, err
	}

	namespaces := make(map[string]*namespace)
	reqs := make(map[string]*datastore.StreamBuffer)
	return &client{clientset, cache, sync.Mutex{}, namespaces, sync.Mutex{}, reqs, time.Now(), name}, nil
}

// Find namespace.
func (cli *client) Find(ctx context.Context, name string) (plugin.Node, error) {
	cli.refreshNamespaces(ctx)
	if ns, ok := cli.namespaces[name]; ok {
		return plugin.NewDir(ns), nil
	}
	return nil, plugin.ENOENT
}

// List all namespaces.
func (cli *client) List(ctx context.Context) ([]plugin.Node, error) {
	cli.refreshNamespaces(ctx)
	log.Debugf("Listing %v namespaces in /kubernetes", len(cli.namespaces))
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
	for _, v := range cli.reqs {
		if updated := v.LastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *client) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

func (cli *client) refreshNamespaces(ctx context.Context) error {
	cli.nsmux.Lock()
	defer cli.nsmux.Unlock()
	namespaces, err := cli.cachedNamespaceList(ctx)
	if err != nil {
		return err
	}

	// Remove unnamed namespaces
	for name := range cli.namespaces {
		if name == allNamespace {
			// Don't remove 'all' namespace.
			continue
		}
		idx := sort.SearchStrings(namespaces, name)
		if namespaces[idx] != name {
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

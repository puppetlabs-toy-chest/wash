package kubernetes

import (
	"context"
	"flag"
	"log"
	"os/user"
	"path/filepath"
	"sort"
	"time"

	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/plugin"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Loads the gcp plugin (required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Client is a kubernetes client.
type Client struct {
	*k8s.Clientset
	cache   *bigcache.BigCache
	debug   bool
	updated time.Time
	root    string
	groups  []string
}

// Create a new kubernetes client.
func Create(debug bool) (*Client, error) {
	me, err := user.Current()
	if err != nil {
		return nil, err
	}

	var kubeconfig *string
	if me.HomeDir != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(me.HomeDir, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
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

	groups := []string{"namespaces", "pods"}
	sort.Strings(groups)

	return &Client{clientset, cache, debug, time.Now(), "kubernetes", groups}, nil
}

func (cli *Client) log(format string, v ...interface{}) {
	if cli.debug {
		log.Printf(format, v...)
	}
}

// Find container by ID.
func (cli *Client) Find(ctx context.Context, parent *plugin.Dir, name string) (plugin.Node, error) {
	switch parent.Name {
	case "kubernetes":
		idx := sort.SearchStrings(cli.groups, name)
		if cli.groups[idx] == name {
			cli.log("Found group %v", name)
			return &plugin.Dir{Client: cli, Parent: parent, Name: name}, nil
		}
	case "pods":
		if pod, err := cli.cachedPodFind(ctx, name); err == nil {
			cli.log("Found pod %v, %v", name, pod)
			return &plugin.File{Client: cli, Parent: parent, Name: name}, nil
		}
	case "namespaces":
		if namespace, err := cli.cachedNamespaceFind(ctx, name); err == nil {
			cli.log("Found namespace %v, %v", name, namespace)
			return &plugin.Dir{Client: cli, Parent: parent, Name: name}, nil
		}
	}

	if parent.Parent.Name == "namespaces" {
		if pods, err := cli.cachedNamespaceFind(ctx, parent.Name); err == nil {
			idx := sort.SearchStrings(pods, name)
			if pods[idx] == name {
				cli.log("Found pod %v in namespace %v", name, parent.Name)
				return &plugin.File{Client: cli, Parent: parent, Name: name}, nil
			}
		}
	}
	return nil, plugin.ENOENT
}

// List all running pods as files.
func (cli *Client) List(ctx context.Context, parent *plugin.Dir) ([]plugin.Node, error) {
	switch parent.Name {
	case "kubernetes":
		cli.log("Listing %v groups in /kubernetes", len(cli.groups))
		entries := make([]plugin.Node, len(cli.groups))
		for i, v := range cli.groups {
			entries[i] = &plugin.Dir{Client: cli, Parent: parent, Name: v}
		}
		return entries, nil
	case "pods":
		pods, err := cli.cachedPodList(ctx)
		if err != nil {
			return nil, err
		}
		cli.log("Listing %v pods in /kubernetes/pods", len(pods))
		entries := make([]plugin.Node, len(pods))
		for i, v := range pods {
			entries[i] = &plugin.File{Client: cli, Parent: parent, Name: v}
		}
		return entries, nil
	case "namespaces":
		namespaces, err := cli.cachedNamespaceList(ctx)
		if err != nil {
			return nil, err
		}
		cli.log("Listing %v namespaces in /kubernetes/namespaces", len(namespaces))
		entries := make([]plugin.Node, len(namespaces))
		for i, v := range namespaces {
			entries[i] = &plugin.Dir{Client: cli, Parent: parent, Name: v}
		}
		return entries, nil
	}

	if parent.Parent.Name == "namespaces" {
		pods, err := cli.cachedNamespaceFind(ctx, parent.Name)
		if err != nil {
			return nil, err
		}
		cli.log("Listing %v pods in /kubernetes/namespaces/%v", len(pods), parent.Name)
		entries := make([]plugin.Node, len(pods))
		for i, v := range pods {
			entries[i] = &plugin.File{Client: cli, Parent: parent, Name: v}
		}
		return entries, nil
	}
	return []plugin.Node{}, nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, name string) (*plugin.Attributes, error) {
	cli.log("Reading attributes of %v in /kubernetes", name)
	return &plugin.Attributes{Mtime: cli.updated, Valid: 1 * time.Second}, nil
}

// Xattr returns a map of extended attributes.
func (cli *Client) Xattr(ctx context.Context, name string) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, name string) (plugin.IFileBuffer, error) {
	return nil, plugin.ENOTSUP
}

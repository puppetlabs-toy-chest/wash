package kubernetes

import (
	"context"
	"flag"
	"log"
	"os/user"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/datastore"
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
	mux     sync.Mutex
	reqs    map[string]*datastore.StreamBuffer
	updated time.Time
	root    string
	groups  []string
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond

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

	reqs := make(map[string]*datastore.StreamBuffer)
	return &Client{clientset, cache, debug, sync.Mutex{}, reqs, time.Now(), "kubernetes", groups}, nil
}

func (cli *Client) log(format string, v ...interface{}) {
	if cli.debug {
		log.Printf(format, v...)
	}
}

// Find container by ID.
func (cli *Client) Find(ctx context.Context, parent *plugin.Dir, name string) (plugin.Entry, error) {
	switch parent.Name() {
	case "kubernetes":
		idx := sort.SearchStrings(cli.groups, name)
		if cli.groups[idx] == name {
			cli.log("Found group %v", name)
			return plugin.NewDir(cli, parent, name), nil
		}
	case "pods":
		if pod, err := cli.cachedPodFind(ctx, name); err == nil {
			cli.log("Found pod %v, %v", name, pod)
			return plugin.NewFile(cli, parent, name), nil
		}
	case "namespaces":
		if namespace, err := cli.cachedNamespaceFind(ctx, name); err == nil {
			cli.log("Found namespace %v, %v", name, namespace)
			return plugin.NewDir(cli, parent, name), nil
		}
	}

	if parent.Parent().Name() == "namespaces" {
		if pods, err := cli.cachedNamespaceFind(ctx, parent.Name()); err == nil {
			idx := sort.SearchStrings(pods, name)
			if pods[idx] == name {
				cli.log("Found pod %v in namespace %v", name, parent.Name())
				return plugin.NewFile(cli, parent, name), nil
			}
		}
	}
	return nil, plugin.ENOENT
}

// List all running pods as files.
func (cli *Client) List(ctx context.Context, parent *plugin.Dir) ([]plugin.Entry, error) {
	switch parent.Name() {
	case "kubernetes":
		cli.log("Listing %v groups in /kubernetes", len(cli.groups))
		entries := make([]plugin.Entry, len(cli.groups))
		for i, v := range cli.groups {
			entries[i] = plugin.NewDir(cli, parent, v)
		}
		return entries, nil
	case "pods":
		pods, err := cli.cachedPodList(ctx)
		if err != nil {
			return nil, err
		}
		cli.log("Listing %v pods in /kubernetes/pods", len(pods))
		entries := make([]plugin.Entry, len(pods))
		for i, v := range pods {
			entries[i] = plugin.NewFile(cli, parent, v)
		}
		return entries, nil
	case "namespaces":
		namespaces, err := cli.cachedNamespaceList(ctx)
		if err != nil {
			return nil, err
		}
		cli.log("Listing %v namespaces in /kubernetes/namespaces", len(namespaces))
		entries := make([]plugin.Entry, len(namespaces))
		for i, v := range namespaces {
			entries[i] = plugin.NewDir(cli, parent, v)
		}
		return entries, nil
	}

	if parent.Parent().Name() == "namespaces" {
		pods, err := cli.cachedNamespaceFind(ctx, parent.Name())
		if err != nil {
			return nil, err
		}
		cli.log("Listing %v pods in /kubernetes/namespaces/%v", len(pods), parent.Name())
		entries := make([]plugin.Entry, len(pods))
		for i, v := range pods {
			entries[i] = plugin.NewFile(cli, parent, v)
		}
		return entries, nil
	}
	return []plugin.Entry{}, nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, node plugin.Entry) (*plugin.Attributes, error) {
	if node == nil || node.Name() == cli.root {
		// Now that content updates are asynchronous, we can make directory mtime reflect when we get new content.
		latest := cli.updated
		for _, v := range cli.reqs {
			if updated := v.LastUpdate(); updated.After(latest) {
				latest = updated
			}
		}
		return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
	}

	cli.log("Reading attributes of %v in /kubernetes", node.Name())
	// Read the content to figure out how large it is.
	cli.mux.Lock()
	defer cli.mux.Unlock()
	if buf, ok := cli.reqs[node.Name()]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *Client) Xattr(ctx context.Context, node plugin.Entry) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, node plugin.Entry) (plugin.IFileBuffer, error) {
	return nil, plugin.ENOTSUP
}

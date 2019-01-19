package kubernetes

import (
	"bytes"
	"context"
	"encoding/gob"
	"flag"
	"log"
	"os/user"
	"path/filepath"
	"time"

	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	return &Client{clientset, cache, debug, time.Now()}, nil
}

func (cli *Client) log(format string, v ...interface{}) {
	if cli.debug {
		log.Printf(format, v...)
	}
}

func (cli *Client) cachedPodList(ctx context.Context) (map[string]corev1.Pod, error) {
	entry, err := cli.cache.Get("PodList")
	var pods map[string]corev1.Pod
	if err == nil {
		cli.log("Cache hit in /kubernetes")
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&pods)
	} else {
		cli.log("Cache miss in /kubernetes")

		podList, err := cli.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		pods = make(map[string]corev1.Pod)
		for _, pod := range podList.Items {
			pods[string(pod.UID)] = pod
		}

		var data bytes.Buffer
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(&pods); err != nil {
			return nil, err
		}
		cli.cache.Set("PodList", data.Bytes())
		cli.updated = time.Now()
	}
	return pods, err
}

// Find container by ID.
func (cli *Client) Find(ctx context.Context, name string) (*plugin.Entry, error) {
	pods, err := cli.cachedPodList(ctx)
	if err != nil {
		return nil, err
	}

	if pod, ok := pods[name]; ok {
		cli.log("Found pod %v, %v", name, pod)
		return &plugin.Entry{Client: cli, Name: name}, nil
	}
	cli.log("Pod %v not found", name)
	return nil, plugin.ENOENT
}

// List all running pods as files.
func (cli *Client) List(ctx context.Context) ([]plugin.Entry, error) {
	pods, err := cli.cachedPodList(ctx)
	if err != nil {
		return nil, err
	}
	cli.log("Listing %v pods in /kubernetes", len(pods))
	keys := make([]plugin.Entry, 0, len(pods))
	for _, pod := range pods {
		keys = append(keys, plugin.Entry{Client: cli, Name: string(pod.UID)})
	}
	return keys, nil
}

// Attr returns attributes of the named resource.
func (cli *Client) Attr(ctx context.Context, name string) (*plugin.Attributes, error) {
	cli.log("Reading attributes of %v in /kubernetes", name)
	return &plugin.Attributes{Mtime: cli.updated}, nil
}

// Xattr returns a map of extended attributes.
func (cli *Client) Xattr(ctx context.Context, name string) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

// Open gets logs from a container.
func (cli *Client) Open(ctx context.Context, name string) (plugin.IFileBuffer, error) {
	return nil, plugin.ENOTSUP
}

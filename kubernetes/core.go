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

func (cli *Client) updateCachedPods(ctx context.Context) error {
	podList, err := cli.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var data bytes.Buffer

	podNames := make([]string, len(podList.Items))
	for i, pod := range podList.Items {
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(pod); err != nil {
			return err
		}
		cli.cache.Set(string(pod.UID), data.Bytes())
		data.Reset()
		podNames[i] = string(pod.UID)
	}

	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&podNames); err != nil {
		return err
	}
	cli.cache.Set("PodList", data.Bytes())

	return nil
}

func (cli *Client) cachedPodList(ctx context.Context) ([]string, error) {
	entry, err := cli.cache.Get("PodList")
	if err != nil {
		cli.log("Cache miss in /kubernetes")
		if err := cli.updateCachedPods(ctx); err != nil {
			return nil, err
		}
		if entry, err = cli.cache.Get("PodList"); err != nil {
			return nil, err
		}
	} else {
		cli.log("Cache hit in /kubernetes")
	}

	var pods []string
	dec := gob.NewDecoder(bytes.NewReader(entry))
	err = dec.Decode(&pods)
	return pods, err
}

func (cli *Client) cachedPodFind(ctx context.Context, name string) (*corev1.Pod, error) {
	entry, err := cli.cache.Get(name)
	if err != nil {
		// If name wasn't found, check whether PodList was loaded and if not load it.
		if _, err := cli.cache.Get("PodList"); err != nil {
			cli.log("Cache miss in /kubernetes/%v", name)
			if err := cli.updateCachedPods(ctx); err != nil {
				return nil, err
			}
			entry, err = cli.cache.Get(name)
		}

		if err != nil {
			// Name wasn't found after refreshing PodList cache (or cache was up-to-date).
			return nil, err
		}
	} else {
		cli.log("Cache hit in /kubernetes/%v", name)
	}

	var pod corev1.Pod
	dec := gob.NewDecoder(bytes.NewReader(entry))
	err = dec.Decode(&pod)
	return &pod, err
}

// Find container by ID.
func (cli *Client) Find(ctx context.Context, name string) (*plugin.Entry, error) {
	if pod, err := cli.cachedPodFind(ctx, name); err == nil {
		cli.log("Found pod %v, %v", name, pod)
		return &plugin.Entry{Client: cli, Name: name}, nil
	} else {
		cli.log("Pod %v not found: %v", name, err)
		return nil, plugin.ENOENT
	}
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
		keys = append(keys, plugin.Entry{Client: cli, Name: pod})
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

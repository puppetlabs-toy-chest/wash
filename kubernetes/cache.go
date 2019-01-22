package kubernetes

import (
	"bytes"
	"context"
	"encoding/gob"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	podPrefix    = "Pod:"
	podCacheName = "PodList"
	nsPrefix     = "NS:"
	nsCacheName  = "Namespaces"
)

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
		cli.cache.Set(podPrefix+string(pod.UID), data.Bytes())
		data.Reset()
		podNames[i] = string(pod.UID)
	}

	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&podNames); err != nil {
		return err
	}
	cli.cache.Set(podCacheName, data.Bytes())

	return nil
}

func (cli *Client) cachedPodList(ctx context.Context) ([]string, error) {
	entry, err := cli.cache.Get(podCacheName)
	if err != nil {
		cli.log("Cache miss in /kubernetes")
		if err := cli.updateCachedPods(ctx); err != nil {
			return nil, err
		}
		if entry, err = cli.cache.Get(podCacheName); err != nil {
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
	entry, err := cli.cache.Get(podPrefix + name)
	if err != nil {
		// If name wasn't found, check whether PodList was loaded and if not load it.
		if _, cerr := cli.cache.Get(podCacheName); cerr != nil {
			cli.log("Cache miss in /kubernetes/%v", name)
			if err = cli.updateCachedPods(ctx); err != nil {
				return nil, err
			}
			if entry, err = cli.cache.Get(podPrefix + name); err != nil {
				return nil, err
			}
		} else {
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

func (cli *Client) updateCachedNamespaces(ctx context.Context) error {
	namespaces, err := cli.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var data bytes.Buffer

	namespaceNames := make([]string, len(namespaces.Items))
	for i, namespace := range namespaces.Items {
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(namespace); err != nil {
			return err
		}
		cli.cache.Set(nsPrefix+namespace.Name, data.Bytes())
		data.Reset()
		namespaceNames[i] = namespace.Name
	}

	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&namespaceNames); err != nil {
		return err
	}
	cli.cache.Set(nsCacheName, data.Bytes())

	return nil
}

func (cli *Client) cachedNamespaceList(ctx context.Context) ([]string, error) {
	entry, err := cli.cache.Get(nsCacheName)
	if err != nil {
		cli.log("Cache miss in /kubernetes")
		if err := cli.updateCachedNamespaces(ctx); err != nil {
			return nil, err
		}
		if entry, err = cli.cache.Get(nsCacheName); err != nil {
			return nil, err
		}
	} else {
		cli.log("Cache hit in /kubernetes")
	}

	var namespaces []string
	dec := gob.NewDecoder(bytes.NewReader(entry))
	err = dec.Decode(&namespaces)
	return namespaces, err
}

func (cli *Client) cachedNamespaceFind(ctx context.Context, name string) (*corev1.Namespace, error) {
	entry, err := cli.cache.Get(nsPrefix + name)
	if err != nil {
		// If name wasn't found, check whether Namespaces was loaded and if not load it.
		if _, cerr := cli.cache.Get(nsCacheName); cerr != nil {
			cli.log("Cache miss in /kubernetes/%v", name)
			if err = cli.updateCachedNamespaces(ctx); err != nil {
				return nil, err
			}
			if entry, err = cli.cache.Get(nsPrefix + name); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		cli.log("Cache hit in /kubernetes/%v", name)
	}

	var namespace corev1.Namespace
	dec := gob.NewDecoder(bytes.NewReader(entry))
	err = dec.Decode(&namespace)
	return &namespace, err
}

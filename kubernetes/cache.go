package kubernetes

import (
	"bytes"
	"context"
	"encoding/gob"
	"sort"
	"time"

	"github.com/puppetlabs/wash/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	podPrefix    = "Pod:"
	podCacheName = "PodList"
	nsPrefix     = "NS:"
	nsCacheName  = "Namespaces"
)

func (cli *client) updateCachedPods(ctx context.Context) error {
	podList, err := cli.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	podNames := make([]string, len(podList.Items))
	namespaces := make(map[string][]string)
	for i, pod := range podList.Items {
		data, err := pod.Marshal()
		if err != nil {
			return err
		}
		cli.cache.Set(podPrefix+string(pod.Name), data)

		podNames[i] = pod.Name
		namespaces[pod.Namespace] = append(namespaces[pod.Namespace], pod.Name)
	}

	var data bytes.Buffer

	namespaceNames := make([]string, 0, len(namespaces))
	for key, value := range namespaces {
		sort.Strings(value)
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(value); err != nil {
			return err
		}
		cli.cache.Set(nsPrefix+key, data.Bytes())
		data.Reset()

		namespaceNames = append(namespaceNames, key)
	}
	sort.Strings(namespaceNames)

	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&podNames); err != nil {
		return err
	}
	cli.cache.Set(podCacheName, data.Bytes())
	data.Reset()

	enc = gob.NewEncoder(&data)
	if err := enc.Encode(&namespaceNames); err != nil {
		return err
	}
	cli.cache.Set(nsCacheName, data.Bytes())
	data.Reset()

	cli.updated = time.Now()
	return nil
}

func (cli *client) retry(ctx context.Context, key string) ([]byte, error) {
	if err := cli.updateCachedPods(ctx); err != nil {
		return nil, err
	}
	return cli.cache.Get(key)
}

func (cli *client) cachedStrings(ctx context.Context, key string, cb func(context.Context, string) ([]byte, error)) ([]string, error) {
	entry, err := cli.cache.Get(key)
	if err != nil {
		log.Printf("Cache miss on kubernetes/%v", key)
		if entry, err = cb(ctx, key); err != nil {
			return nil, err
		}
	} else {
		log.Debugf("Cache hit on kubernetes/%v", key)
	}

	var strings []string
	dec := gob.NewDecoder(bytes.NewReader(entry))
	err = dec.Decode(&strings)
	return strings, err
}

func (cli *client) cachedPodList(ctx context.Context) ([]string, error) {
	return cli.cachedStrings(ctx, podCacheName, cli.retry)
}

func (cli *client) cachedNamespaceList(ctx context.Context) ([]string, error) {
	return cli.cachedStrings(ctx, nsCacheName, cli.retry)
}

func (cli *client) cachedPodFind(ctx context.Context, name string) (*corev1.Pod, error) {
	entry, err := cli.cache.Get(podPrefix + name)
	if err != nil {
		// If name wasn't found, check whether PodList was loaded and if not load it.
		if _, cerr := cli.cache.Get(podCacheName); cerr != nil {
			log.Debugf("Cache miss in /kubernetes/%v", name)
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
		log.Debugf("Cache hit in /kubernetes/%v", name)
	}

	var pod corev1.Pod
	err = pod.Unmarshal(entry)
	return &pod, err
}

func (cli *client) cachedNamespaceFind(ctx context.Context, name string) ([]string, error) {
	return cli.cachedStrings(ctx, nsPrefix+name, func(ctx context.Context, key string) ([]byte, error) {
		// If name wasn't found, check whether Namespaces was loaded and if not load it.
		if _, err := cli.cache.Get(nsCacheName); err != nil {
			if err := cli.updateCachedPods(ctx); err != nil {
				return nil, err
			}
		}
		return cli.cache.Get(key)
	})
}

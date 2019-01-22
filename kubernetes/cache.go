package kubernetes

import (
	"bytes"
	"context"
	"encoding/gob"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

package kubernetes

import (
	"context"
	"encoding/json"
	"io"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pod struct {
	*resourcetype
	name string
	ns   string
}

func newPod(cli *resourcetype, id string) *pod {
	name, ns := datastore.SplitCompositeString(id)
	return &pod{cli, name, ns}
}

// A unique string describing the pod. Note that the same pod may appear in a specific namespace and 'all'.
// It should use the same identifier in both cases.
func (cli *pod) String() string {
	return cli.resourcetype.client.Name() + "/" + cli.ns + "/pod/" + cli.Name()
}

// Name returns the pod's name.
func (cli *pod) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *pod) Attr(ctx context.Context) (*plugin.Attributes, error) {
	log.Debugf("Reading attributes of %v", cli)
	// Read the content to figure out how large it is.
	if v, ok := cli.reqs.Load(cli.name); ok {
		buf := v.(*datastore.StreamBuffer)
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *pod) Xattr(ctx context.Context) (map[string][]byte, error) {
	// Return metadata for the pod if it's already loaded.
	key := cli.String()
	if entry, err := cli.resourcetype.client.cache.Get(key); err != nil {
		log.Printf("Cache miss on %v, skipping lookup", key)
	} else {
		log.Debugf("Cache hit on %v", key)
		return plugin.JSONToJSONMap(entry)
	}
	return map[string][]byte{}, nil
}

func (cli *pod) readLog() (io.ReadCloser, error) {
	opts := corev1.PodLogOptions{
		Follow: true,
	}
	req := cli.CoreV1().Pods(cli.ns).GetLogs(cli.name, &opts)
	return req.Stream()
}

// Open gets logs from a container.
func (cli *pod) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: store as UID? Names are not guaranteed to be unique across namespaces, so the pods/ list may
	// include duplicates. We can fix that by using UIDs or an amalgam of namespace+name, but then we have
	// to always map that to the namespace and name when loading logs and make sure attribute queries always
	// use a consistent key for lookup.
	buf := datastore.NewBuffer(cli.name, nil)
	if v, ok := cli.reqs.LoadOrStore(cli.name, buf); ok {
		buf = v.(*datastore.StreamBuffer)
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(cli.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return buf, nil
}

func (cli *client) cachedPods(ctx context.Context, ns string) ([]string, error) {
	return datastore.CachedStrings(cli.cache, cli.Name()+"/pods/"+ns, func() ([]string, error) {
		// Query all pods and refresh all cache entries. Then return just the one that was requested.
		podList, err := cli.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		allpods := make([]string, len(podList.Items))
		pods := make(map[string][]string)
		for i, pd := range podList.Items {
			allpods[i] = datastore.MakeCompositeString(pd.Name, pd.Namespace)
			pods[pd.Namespace] = append(pods[pd.Namespace], allpods[i])
			// Also cache individual pod data as JSON for use in xattributes.
			if bits, err := json.Marshal(pd); err == nil {
				cli.cache.Set(cli.Name()+"/"+pd.Namespace+"/pod/"+pd.Name, bits)
			} else {
				log.Printf("Unable to marshal pod %v: %v", pd, err)
			}
		}
		pods[allNamespace] = allpods

		for name, data := range pods {
			// Skip the one we're returning because CachedStrings will encode and store to cache for us.
			if name != ns {
				datastore.CacheAny(cli.cache, cli.Name()+"/pods/"+name, data)
			}
		}
		return pods[ns], nil
	})
}

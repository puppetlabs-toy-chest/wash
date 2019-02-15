package kubernetes

import (
	"context"
	"encoding/json"
	"io"

	"github.com/puppetlabs/wash/datastore"
	log "github.com/sirupsen/logrus"
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
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size())}, nil
	}

	// Prefetch content for next time.
	go plugin.PrefetchOpen(cli)

	return &plugin.Attributes{Mtime: cli.updated}, nil
}

// Xattr returns a map of extended attributes.
func (cli *pod) Xattr(ctx context.Context) (map[string][]byte, error) {
	entry, err := cli.cache.CachedJSON(cli.String(), func() ([]byte, error) {
		pd, err := cli.CoreV1().Pods(cli.ns).Get(cli.name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return json.Marshal(pd)
	})
	if err != nil {
		return nil, err
	}
	return plugin.JSONToJSONMap(entry)
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
	return cli.cache.CachedStrings(cli.Name()+"/pods/"+ns, func() ([]string, error) {
		// Query pods and refresh all cache entries. Then return just the one that was requested.
		podList, err := cli.CoreV1().Pods(cli.queryScope()).List(metav1.ListOptions{})
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
				log.Infof("Unable to marshal pod %v: %v", pd, err)
			}
		}
		pods[allNamespace] = allpods

		for name, data := range pods {
			// Skip the one we're returning because CachedStrings will encode and store to cache for us.
			if name != ns {
				cli.cache.SetAny(cli.Name()+"/pods/"+name, data, datastore.Fast)
			}
		}
		return pods[ns], nil
	})
}

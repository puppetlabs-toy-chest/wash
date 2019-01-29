package kubernetes

import (
	"context"
	"encoding/json"
	"io"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	corev1 "k8s.io/api/core/v1"
)

type pod struct {
	*client
	name string
}

// Name returns the pod's name.
func (cli *pod) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *pod) Attr(ctx context.Context) (*plugin.Attributes, error) {
	log.Debugf("Reading attributes of %v in /kubernetes", cli.name)
	// Read the content to figure out how large it is.
	cli.mux.Lock()
	defer cli.mux.Unlock()
	if buf, ok := cli.reqs[cli.name]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *pod) Xattr(ctx context.Context) (map[string][]byte, error) {
	pod, err := cli.cachedPodFind(ctx, cli.name)
	if err != nil {
		return nil, err
	}

	inrec, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	return plugin.JSONToJSONMap(inrec)
}

func (cli *pod) readLog(name string) (io.ReadCloser, error) {
	// Pod logs must be retrieved by namespace and name, not UID.
	pod, err := cli.cachedPodFind(context.Background(), name)
	if err != nil {
		return nil, err
	}

	opts := corev1.PodLogOptions{
		Follow: true,
	}
	req := cli.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &opts)
	return req.Stream()
}

// Open gets logs from a container.
func (cli *pod) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	cli.mux.Lock()
	defer cli.mux.Unlock()

	// TODO: store as UID? Names are not guaranteed to be unique across namespaces, so the pods/ list may
	// include duplicates. We can fix that by using UIDs or an amalgam of namespace+name, but then we have
	// to always map that to the namespace and name when loading logs and make sure attribute queries always
	// use a consistent key for lookup.
	buf, ok := cli.reqs[cli.name]
	if !ok {
		buf = datastore.NewBuffer(cli.name, nil)
		cli.reqs[cli.name] = buf
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(cli.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return buf, nil
}

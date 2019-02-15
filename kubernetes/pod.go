package kubernetes

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type pod struct {
	plugin.EntryBase
	podi      typedv1.PodInterface
	startTime time.Time
	meta      datastore.Var
}

func newPod(pi typedv1.PodInterface, p *corev1.Pod) *pod {
	pd := &pod{
		EntryBase: plugin.NewEntry(p.Name),
		podi:      pi,
		startTime: p.CreationTimestamp.Time,
		meta:      datastore.NewVar(5 * time.Second),
	}
	pd.meta.Set(plugin.ToMetadata(p))
	return pd
}

func (p *pod) Metadata(ctx context.Context) (map[string]interface{}, error) {
	meta, err := p.meta.Update(func() (interface{}, error) {
		pd, err := p.podi.Get(p.Name(), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return plugin.ToMetadata(pd), nil
	})
	if err != nil {
		return nil, err
	}
	return meta.(map[string]interface{}), nil
}

func (p *pod) Attr() plugin.Attributes {
	return plugin.Attributes{
		Ctime: p.startTime,
		Mtime: time.Now(),
		Atime: p.startTime,
		Size:  plugin.SizeUnknown,
	}
}

func (p *pod) Open(ctx context.Context) (plugin.SizedReader, error) {
	req := p.podi.GetLogs(p.Name(), &corev1.PodLogOptions{})
	rdr, err := req.Stream()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	var n int64
	if n, err = buf.ReadFrom(rdr); err != nil {
		return nil, err
	}
	log.Debugf("Read %v bytes of %v log", n, p.Name())
	return bytes.NewReader(buf.Bytes()), nil
}

func (p *pod) Stream(ctx context.Context) (io.Reader, error) {
	var tailLines int64 = 10
	req := p.podi.GetLogs(p.Name(), &corev1.PodLogOptions{Follow: true, TailLines: &tailLines})
	return req.Stream()
}

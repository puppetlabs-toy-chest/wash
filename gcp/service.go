package gcp

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/pubsub"
	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	dataflow "google.golang.org/api/dataflow/v1b3"
)

type service struct {
	name    string
	proj    string
	updated time.Time
	client  interface{}
	mux     sync.Mutex
	reqs    map[string]*datastore.StreamBuffer
	cache   *bigcache.BigCache
}

// Google auto-generated API scopes needed by services.
var serviceScopes = []string{dataflow.CloudPlatformScope}

func buffer() map[string]*datastore.StreamBuffer {
	return make(map[string]*datastore.StreamBuffer)
}

func newServices(projectName string, oauthClient *http.Client, cache *bigcache.BigCache) (map[string]*service, error) {
	ongoing := context.Background()
	pubsubClient, err := pubsub.NewClient(ongoing, projectName)
	if err != nil {
		return nil, err
	}

	dataflowClient, err := dataflow.New(oauthClient)
	if err != nil {
		return nil, err
	}

	bigqueryClient, err := bigquery.NewClient(ongoing, projectName)
	if err != nil {
		return nil, err
	}

	services := make(map[string]*service)
	for name, client := range map[string]interface{}{
		"pubsub":   pubsubClient,
		"dataflow": dataflowClient,
		"bigquery": bigqueryClient,
	} {
		services[name] = &service{name, projectName, time.Now(), client, sync.Mutex{}, buffer(), cache}
	}
	return services, nil
}

// Find resource by name.
func (cli *service) Find(ctx context.Context, name string) (plugin.Node, error) {
	switch c := cli.client.(type) {
	case *pubsub.Client:
		topics, err := cli.cachedTopics(ctx, c)
		if err != nil {
			return nil, err
		}

		idx := sort.SearchStrings(topics, name)
		if topics[idx] == name {
			return plugin.NewFile(&pubsubTopic{name, c, cli}), nil
		}
		return nil, plugin.ENOENT
	case *dataflow.Service:
		jobs, err := cli.cachedDataflowJobs(c)
		if err != nil {
			return nil, err
		}

		if id, ok := searchDataflowJob(jobs, name); ok {
			return plugin.NewFile(newDataflowJob(id, c, cli)), nil
		}
		return nil, plugin.ENOENT
	case *bigquery.Client:
		datasets, err := cli.cachedDatasets(ctx, c)
		if err != nil {
			return nil, err
		}

		idx := sort.SearchStrings(datasets, name)
		if datasets[idx] == name {
			return plugin.NewDir(newBigqueryDataset(name, c, cli)), nil
		}
		return nil, plugin.ENOENT
	}
	return nil, plugin.ENOTSUP
}

// List all resources as files/dirs.
func (cli *service) List(ctx context.Context) ([]plugin.Node, error) {
	switch c := cli.client.(type) {
	case *pubsub.Client:
		topics, err := cli.cachedTopics(ctx, c)
		if err != nil {
			return nil, err
		}
		entries := make([]plugin.Node, len(topics))
		for i, id := range topics {
			entries[i] = plugin.NewFile(&pubsubTopic{id, c, cli})
		}
		return entries, nil
	case *dataflow.Service:
		jobs, err := cli.cachedDataflowJobs(c)
		if err != nil {
			return nil, err
		}
		entries := make([]plugin.Node, len(jobs))
		for i, id := range jobs {
			entries[i] = plugin.NewFile(newDataflowJob(id, c, cli))
		}
		return entries, nil
	case *bigquery.Client:
		datasets, err := cli.cachedDatasets(ctx, c)
		if err != nil {
			return nil, err
		}
		entries := make([]plugin.Node, len(datasets))
		for i, name := range datasets {
			entries[i] = plugin.NewDir(newBigqueryDataset(name, c, cli))
		}
		return entries, nil
	}
	return nil, plugin.ENOTSUP
}

// String returns a printable representation of the service.
func (cli *service) String() string {
	return fmt.Sprintf("gcp/%v/%v", cli.proj, cli.name)
}

// Name returns the service name.
func (cli *service) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *service) Attr(ctx context.Context) (*plugin.Attributes, error) {
	return &plugin.Attributes{Mtime: cli.lastUpdate(), Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *service) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

func (cli *service) close(ctx context.Context) error {
	switch c := cli.client.(type) {
	case *pubsub.Client:
		return c.Close()
	}
	return plugin.ENOTSUP
}

func (cli *service) lastUpdate() time.Time {
	latest := cli.updated
	for _, v := range cli.reqs {
		if updated := v.LastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return latest
}

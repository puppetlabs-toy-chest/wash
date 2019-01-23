package gcp

import (
	"bytes"
	"context"
	"encoding/gob"
	"sort"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type service struct {
	name    string
	proj    string
	updated time.Time
	client  interface{}
	reqs    map[string]*datastore.StreamBuffer
	cache   *bigcache.BigCache
}

func newServices(projectName string, cache *bigcache.BigCache) (map[string]*service, error) {
	pubsub, err := pubsub.NewClient(context.Background(), projectName)
	if err != nil {
		return nil, err
	}
	reqs := make(map[string]*datastore.StreamBuffer)
	pubsubService := &service{"pubsub", projectName, time.Now(), pubsub, reqs, cache}

	return map[string]*service{
		"pubsub": pubsubService,
	}, nil
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
			return plugin.NewFile(&topic{name, c, cli}), nil
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
			entries[i] = plugin.NewFile(&topic{id, c, cli})
		}
		return entries, nil
	}
	return nil, plugin.ENOTSUP
}

// Returns the service name.
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

func (cli *service) cachedTopics(ctx context.Context, c *pubsub.Client) ([]string, error) {
	key := cli.proj + "/" + cli.name
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit in /gcp")
		var topics []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&topics)
		return topics, err
	}

	log.Debugf("Cache miss in /gcp")
	topics := make([]string, 0)
	it := c.Topics(ctx)
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		topics = append(topics, t.ID())
	}
	sort.Strings(topics)

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&topics); err != nil {
		return nil, err
	}
	cli.cache.Set(key, data.Bytes())
	cli.updated = time.Now()
	return topics, nil
}

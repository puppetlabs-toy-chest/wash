package gcp

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type pubsubTopic struct {
	name   string
	client *pubsub.Client
	*service
}

// String returns a unique representation of the pubsubTopic.
func (cli *pubsubTopic) String() string {
	return cli.service.String() + "/" + cli.Name()
}

// Returns the pubsubTopic name.
func (cli *pubsubTopic) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *pubsubTopic) Attr(ctx context.Context) (*plugin.Attributes, error) {
	if v, ok := cli.reqs.Load(cli.name); ok {
		buf := v.(*datastore.StreamBuffer)
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size())}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated}, nil
}

// Xattr returns a map of extended attributes.
func (cli *pubsubTopic) Xattr(ctx context.Context) (map[string][]byte, error) {
	tpc := cli.client.Topic(cli.name)
	data := make(map[string][]byte)
	topicConfig, err := tpc.Config(ctx)
	if err != nil {
		return nil, err
	}

	data["Labels"], err = json.Marshal(topicConfig.Labels)
	if err != nil {
		return nil, err
	}

	data["MessageStoragePolicy"], err = json.Marshal(topicConfig.MessageStoragePolicy)
	if err != nil {
		return nil, err
	}

	subs := make([]string, 0)
	it := tpc.Subscriptions(ctx)
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		subs = append(subs, s.ID())
	}
	data["Subscriptions"], err = json.Marshal(subs)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Open subscribes to a pubsubTopic and reads new messages.
func (cli *pubsubTopic) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: subscribe to pubsubTopic when opened.
	// https://godoc.org/cloud.google.com/go/pubsub#Client.CreateSubscription
	// https://godoc.org/cloud.google.com/go/pubsub#Subscription.Receive
	return nil, plugin.ENOTSUP
}

func (cli *service) cachedTopics(ctx context.Context, c *pubsub.Client) ([]string, error) {
	return cli.cache.CachedStrings(cli.String(), func() ([]string, error) {
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
		return topics, nil
	})
}

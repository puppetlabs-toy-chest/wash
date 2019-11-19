package gcp

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type pubsubDir struct {
	plugin.EntryBase
	client *pubsub.Client
}

func newPubsubDir(ctx context.Context, projID string) (*pubsubDir, error) {
	cli, err := pubsub.NewClient(context.Background(), projID)
	if err != nil {
		return nil, err
	}
	p := &pubsubDir{
		EntryBase: plugin.NewEntry("pubsub"),
		client:    cli,
	}
	if _, err := plugin.List(ctx, p); err != nil {
		p.MarkInaccessible(ctx, err)
	}
	return p, nil
}

// List all topics as dirs
func (p *pubsubDir) List(ctx context.Context) ([]plugin.Entry, error) {
	topics := make([]plugin.Entry, 0)
	it := p.client.Topics(ctx)
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		topics = append(topics, newPubsubTopic(p.client, t))
	}
	return topics, nil
}

func (p *pubsubDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(p, "pubsub").
		IsSingleton().
		SetDescription(pubsubDirDescription)
}

func (p *pubsubDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&pubsubTopic{}).Schema(),
	}
}

const pubsubDirDescription = `
This directory represents Cloud Pub/Sub. Its entries consist of Pub/Sub topics.

You can publish a message to a topic by appending text to the topic file. For example
		wash gcp/project/pubsub > tail -f topic &
		wash gcp/project/pubsub > echo hello >> topic
		===> my-topic <===
		Nov 21 00:25:14.633 | hello
`

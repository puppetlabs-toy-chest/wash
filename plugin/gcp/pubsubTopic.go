package gcp

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type pubsubTopic struct {
	plugin.EntryBase
	client *pubsub.Client
	topic  *pubsub.Topic
}

type pubsubTopicAttrMeta struct {
	PublishSettings pubsub.PublishSettings
}

type pubsubTopicMetadata struct {
	pubsubTopicAttrMeta
	TopicConfig   pubsub.TopicConfig
	Subscriptions []string
}

func newPubsubTopic(client *pubsub.Client, topic *pubsub.Topic) *pubsubTopic {
	top := &pubsubTopic{
		EntryBase: plugin.NewEntry(topic.ID()),
		client:    client,
		topic:     topic,
	}
	top.
		Attributes().
		SetMeta(pubsubTopicAttrMeta{PublishSettings: topic.PublishSettings})
	return top
}

func (t *pubsubTopic) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	cfg, err := t.topic.Config(ctx)
	if err != nil {
		return nil, err
	}

	subs := make([]string, 0)
	it := t.topic.Subscriptions(ctx)
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

	return plugin.ToJSONObject(pubsubTopicMetadata{
		pubsubTopicAttrMeta: pubsubTopicAttrMeta{PublishSettings: t.topic.PublishSettings},
		TopicConfig:         cfg,
		Subscriptions:       subs,
	}), nil
}

func (t *pubsubTopic) Delete(ctx context.Context) (bool, error) {
	return true, t.topic.Delete(ctx)
}

func (t *pubsubTopic) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(t, "topic").
		SetMetaAttributeSchema(&pubsubTopicAttrMeta{}).
		SetMetadataSchema(&pubsubTopicMetadata{}).
		SetDescription(pubsubTopicDescription)
}

const pubsubTopicDescription = `
A Cloud Pub/Sub topic. You can pipe text to it to publish messages.
`

package gcp

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type pubsubTopic struct {
	plugin.EntryBase
	client *pubsub.Client
	topic  *pubsub.Topic
}

type pubsubTopicPartialMeta struct {
	PublishSettings pubsub.PublishSettings
}

type pubsubTopicMetadata struct {
	pubsubTopicPartialMeta
	TopicConfig   pubsub.TopicConfig
	Subscriptions []string
}

func newPubsubTopic(client *pubsub.Client, topic *pubsub.Topic) *pubsubTopic {
	top := &pubsubTopic{
		EntryBase: plugin.NewEntry(topic.ID()),
		client:    client,
		topic:     topic,
	}
	top.SetPartialMetadata(pubsubTopicPartialMeta{PublishSettings: topic.PublishSettings})

	// This may be somewhat hacky, but it ensures the goroutines for publishing get cleaned up eventually.
	// Writeable can optionally also implement io.Closer to do cleanup when we're done writing,
	// but we may re-use this client so we shouldn't stop it in Close.
	runtime.SetFinalizer(top, func(t *pubsubTopic) { t.topic.Stop() })
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
		pubsubTopicPartialMeta: pubsubTopicPartialMeta{PublishSettings: t.topic.PublishSettings},
		TopicConfig:            cfg,
		Subscriptions:          subs,
	}), nil
}

func (t *pubsubTopic) Delete(ctx context.Context) (bool, error) {
	return true, t.topic.Delete(ctx)
}

// A ReadCloser that subscribes to a topic and buffers all messages that appear there.
type pubsubTopicWatcher struct {
	ctx   context.Context
	sub   *pubsub.Subscription
	queue <-chan *pubsub.Message
	err   <-chan error
}

func (t *pubsubTopic) newPubsubTopicWatcher(ctx context.Context) (*pubsubTopicWatcher, error) {
	sub, err := t.client.CreateSubscription(ctx, "wash-"+uuid.New().String(), pubsub.SubscriptionConfig{
		Topic:            t.topic,
		AckDeadline:      10 * time.Second,
		ExpirationPolicy: 24 * time.Hour,
	})
	if err != nil {
		return nil, err
	}

	// Use a buffer so we can Ack messages quickly.
	queue := make(chan *pubsub.Message, 5)
	errCh := make(chan error)
	watcher := &pubsubTopicWatcher{ctx: ctx, sub: sub, queue: queue, err: errCh}

	bufferMessages := func(_ context.Context, msg *pubsub.Message) {
		msg.Ack()
		queue <- msg
	}
	go func() {
		errCh <- sub.Receive(ctx, bufferMessages)
		close(errCh)
		close(queue)
	}()
	return watcher, nil
}

func (w *pubsubTopicWatcher) Read(p []byte) (int, error) {
	// Wait for an outstanding message, context completion, or error.
	select {
	case <-w.ctx.Done():
		return 0, io.EOF
	case msg := <-w.queue:
		activity.Record(w.ctx, "Reading next message: %v", msg)

		// TODO: don't truncate messages longer than the read buffer.
		s := fmt.Sprintf("%v | %v", msg.PublishTime.Format(time.StampMilli), string(msg.Data))
		return copy(p, []byte(s)), nil
	case err := <-w.err:
		return 0, err
	}
}

func (w *pubsubTopicWatcher) Close() error {
	return w.sub.Delete(context.Background())
}

func (t *pubsubTopic) Stream(ctx context.Context) (io.ReadCloser, error) {
	// Create subscription and wrap it in a ReadCloser to handle cleanup.
	return t.newPubsubTopicWatcher(ctx)
}

func (t *pubsubTopic) Write(ctx context.Context, b []byte) error {
	result := t.topic.Publish(ctx, &pubsub.Message{Data: b})
	sid, err := result.Get(ctx)
	activity.Record(ctx, "Message %v published with server ID %v: %v", string(b), sid, err)
	return err
}

func (t *pubsubTopic) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(t, "topic").
		SetPartialMetadataSchema(&pubsubTopicPartialMeta{}).
		SetMetadataSchema(&pubsubTopicMetadata{}).
		SetDescription(pubsubTopicDescription)
}

const pubsubTopicDescription = `
A Cloud Pub/Sub topic. You can pipe text to it to publish messages.
`

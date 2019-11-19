package gcp

import (
	"container/list"
	"context"
	"fmt"
	"io"
	"sync"
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

// A ReadCloser that subscribes to a topic and buffers all messages that appear there.
// Uses a mutex to synchronize accessing the queue (List) and recording receive errors.
// It shares this mutex because accessing the error is low cost and unlikely to happen
// while we're still buffering messages.
type pubsubTopicWatcher struct {
	ctx   context.Context
	sub   *pubsub.Subscription
	mux   sync.Mutex
	queue *list.List
	err   error
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

	watcher := &pubsubTopicWatcher{ctx: ctx, sub: sub, queue: list.New()}

	bufferMessages := func(_ context.Context, msg *pubsub.Message) {
		msg.Ack()
		watcher.mux.Lock()
		watcher.queue.PushBack(msg)
		watcher.mux.Unlock()
	}
	go func() {
		err := sub.Receive(ctx, bufferMessages)
		watcher.mux.Lock()
		watcher.err = err
		watcher.mux.Unlock()
	}()
	return watcher, nil
}

func (w *pubsubTopicWatcher) Read(p []byte) (int, error) {
	// If there are outstanding messages, return one.
	// If not, check if the context is done before returning.
	var msg *pubsub.Message
	w.mux.Lock()
	if w.queue.Len() > 0 {
		e := w.queue.Front()
		w.queue.Remove(e)
		msg = e.Value.(*pubsub.Message)
	}
	w.mux.Unlock()

	activity.Record(w.ctx, "Reading next message: %v", msg)
	if msg == nil {
		select {
		case <-w.ctx.Done():
			return 0, io.EOF
		default:
			w.mux.Lock()
			defer w.mux.Unlock()
			return 0, w.err
		}
	}

	// TODO: don't truncate messages longer than the read buffer.
	s := fmt.Sprintf("%v | %v", msg.PublishTime.Format(time.StampMilli), string(msg.Data))
	return copy(p, []byte(s)), nil
}

func (w *pubsubTopicWatcher) Close() error {
	return w.sub.Delete(context.Background())
}

func (t *pubsubTopic) Stream(ctx context.Context) (io.ReadCloser, error) {
	// Create subscription and wrap it in a ReadCloser to handle cleanup.
	return t.newPubsubTopicWatcher(ctx)
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

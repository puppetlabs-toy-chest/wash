package gcp

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type topic struct {
	name   string
	client *pubsub.Client
	*service
}

// Returns the topic name.
func (cli *topic) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *topic) Attr(ctx context.Context) (*plugin.Attributes, error) {
	if buf, ok := cli.reqs[cli.name]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *topic) Xattr(ctx context.Context) (map[string][]byte, error) {
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

// Open subscribes to a topic and reads new messages.
func (cli *topic) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: subscribe to topic when opened.
	// https://godoc.org/cloud.google.com/go/pubsub#Client.CreateSubscription
	// https://godoc.org/cloud.google.com/go/pubsub#Subscription.Receive
	return nil, plugin.ENOTSUP
}

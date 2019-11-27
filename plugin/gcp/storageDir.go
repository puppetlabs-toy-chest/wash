package gcp

import (
	"context"
	"net/http"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type storageProjectClient struct {
	*storage.Client
	metrics   *monitoring.MetricClient
	projectID string
}

type storageDir struct {
	plugin.EntryBase
	storageProjectClient
}

const storageScope = storage.ScopeReadOnly

func newStorageDir(ctx context.Context, client *http.Client, projID string) (*storageDir, error) {
	clientContext := context.Background()
	cli, err := storage.NewClient(clientContext, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	metrics, err := monitoring.NewMetricClient(clientContext)
	if err != nil {
		activity.Record(ctx, "Unable to create metrics client for %v/storage: %v", projID, err)
	}

	s := &storageDir{
		EntryBase:            plugin.NewEntry("storage"),
		storageProjectClient: storageProjectClient{Client: cli, metrics: metrics, projectID: projID},
	}
	if _, err := plugin.List(ctx, s); err != nil {
		s.MarkInaccessible(ctx, err)
	}
	return s, nil
}

// List all storage buckets as dirs.
func (s *storageDir) List(ctx context.Context) ([]plugin.Entry, error) {
	var entries []plugin.Entry
	it := s.Buckets(ctx, s.projectID)
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, newStorageBucket(s.storageProjectClient, bucketAttrs))
	}
	return entries, nil
}

func (s *storageDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "storage").IsSingleton()
}

func (s *storageDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{(&storageBucket{}).Schema()}
}

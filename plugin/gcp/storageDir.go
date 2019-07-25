package gcp

import (
	"context"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type storageProjectClient struct {
	*storage.Client
	projectID string
}

type storageDir struct {
	plugin.EntryBase
	storageProjectClient
}

const storageScope = storage.ScopeReadOnly

func newStorageDir(client *http.Client, projID string) (*storageDir, error) {
	cli, err := storage.NewClient(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &storageDir{
		EntryBase:            plugin.NewEntry("storage"),
		storageProjectClient: storageProjectClient{Client: cli, projectID: projID},
	}, nil
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
		entries = append(entries, newStorageBucket(bucketAttrs))
	}
	return entries, nil
}

func (s *storageDir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "storage").IsSingleton()
}

func (s *storageDir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{(&storageBucket{}).Schema()}
}

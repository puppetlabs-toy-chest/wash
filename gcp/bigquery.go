package gcp

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/iterator"
)

type bigqueryDataset struct {
	*bigquery.Dataset
	*service
}

func newBigqueryDataset(name string, c *bigquery.Client, cli *service) *bigqueryDataset {
	return &bigqueryDataset{c.Dataset(name), cli}
}

// String returns a unique representation of the bigquery dataset.
func (cli *bigqueryDataset) String() string {
	return cli.service.String() + "/" + cli.Name()
}

// Returns the bigquery dataset name.
func (cli *bigqueryDataset) Name() string {
	return cli.DatasetID
}

// Find table by name.
func (cli *bigqueryDataset) Find(ctx context.Context, name string) (plugin.Node, error) {
	tables, err := cli.cachedTables(ctx, cli.Dataset)
	if err != nil {
		return nil, err
	}
	if datastore.ContainsString(tables, name) {
		return plugin.NewFile(&bigqueryTable{cli.Table(name), cli}), nil
	}
	return nil, plugin.ENOENT
}

// List all tables as files.
func (cli *bigqueryDataset) List(ctx context.Context) ([]plugin.Node, error) {
	tables, err := cli.cachedTables(ctx, cli.Dataset)
	if err != nil {
		return nil, err
	}
	entries := make([]plugin.Node, len(tables))
	for i, name := range tables {
		entries[i] = plugin.NewFile(&bigqueryTable{cli.Table(name), cli})
	}
	return entries, nil
}

// Attr returns attributes of the named resource.
func (cli *bigqueryDataset) Attr(ctx context.Context) (*plugin.Attributes, error) {
	return &plugin.Attributes{Mtime: cli.updated}, nil
}

// Xattr returns a map of extended attributes.
func (cli *bigqueryDataset) Xattr(ctx context.Context) (map[string][]byte, error) {
	data, err := cli.cache.CachedJSON(cli.String()+"/meta", func() ([]byte, error) {
		meta, err := cli.Metadata(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(meta)
	})
	if err != nil {
		return nil, err
	}
	return plugin.JSONToJSONMap(data)
}

type bigqueryTable struct {
	*bigquery.Table
	dataset *bigqueryDataset
}

// String returns a unique representation of the bigquery table.
func (cli *bigqueryTable) String() string {
	return cli.dataset.String() + "/" + cli.Name()
}

// Returns the bigquery table name.
func (cli *bigqueryTable) Name() string {
	return cli.TableID
}

// Attr returns attributes of the named table.
func (cli *bigqueryTable) Attr(ctx context.Context) (*plugin.Attributes, error) {
	return &plugin.Attributes{Mtime: cli.dataset.updated}, nil
}

// Xattr returns a map of extended attributes.
func (cli *bigqueryTable) Xattr(ctx context.Context) (map[string][]byte, error) {
	data, err := cli.dataset.cache.CachedJSON(cli.String()+"/meta", func() ([]byte, error) {
		meta, err := cli.Metadata(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(meta)
	})
	if err != nil {
		return nil, err
	}
	return plugin.JSONToJSONMap(data)
}

// Open is not supported for bigquery tables.
func (cli *bigqueryTable) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: what content is available from a table? Can we stream insertions?
	return nil, plugin.ENOTSUP
}

func (cli *service) cachedDatasets(ctx context.Context, c *bigquery.Client) ([]string, error) {
	return cli.cache.CachedStrings(cli.String(), func() ([]string, error) {
		datasets := make([]string, 0)
		it := c.Datasets(ctx)
		for {
			d, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			datasets = append(datasets, d.DatasetID)
		}
		cli.updated = time.Now()
		return datasets, nil
	})
}

func (cli *service) cachedTables(ctx context.Context, d *bigquery.Dataset) ([]string, error) {
	return cli.cache.CachedStrings(cli.String()+"/"+d.DatasetID, func() ([]string, error) {
		tables := make([]string, 0)
		it := d.Tables(ctx)
		for {
			t, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			tables = append(tables, t.TableID)
		}
		cli.updated = time.Now()
		return tables, nil
	})
}

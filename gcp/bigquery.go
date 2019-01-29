package gcp

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/puppetlabs/wash/log"
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

// String returns a printable representation of the bigquery dataset.
func (cli *bigqueryDataset) String() string {
	return fmt.Sprintf("gcp/%v/bigquery/%v", cli.ProjectID, cli.DatasetID)
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
	idx := sort.SearchStrings(tables, name)
	if tables[idx] == name {
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
	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *bigqueryDataset) Xattr(ctx context.Context) (map[string][]byte, error) {
	// TODO: return dataset config, https://godoc.org/cloud.google.com/go/bigquery#TableMetadata
	return nil, plugin.ENOTSUP
}

type bigqueryTable struct {
	*bigquery.Table
	dataset *bigqueryDataset
}

// String returns a printable representation of the bigquery table.
func (cli *bigqueryTable) String() string {
	return fmt.Sprintf("gcp/%v/bigquery/%v/%v", cli.ProjectID, cli.DatasetID, cli.TableID)
}

// Returns the bigquery table name.
func (cli *bigqueryTable) Name() string {
	return cli.TableID
}

// Attr returns attributes of the named table.
func (cli *bigqueryTable) Attr(ctx context.Context) (*plugin.Attributes, error) {
	return &plugin.Attributes{Mtime: cli.dataset.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *bigqueryTable) Xattr(ctx context.Context) (map[string][]byte, error) {
	// TODO: return dataset config, https://godoc.org/cloud.google.com/go/bigquery#TableMetadata
	return nil, plugin.ENOTSUP
}

// Open is not supported for bigquery tables.
func (cli *bigqueryTable) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: what content is available from a table? Schema? Can we stream insertions?
	return nil, plugin.ENOTSUP
}

func (cli *service) cachedDatasets(ctx context.Context, c *bigquery.Client) ([]string, error) {
	key := cli.proj + "/" + cli.name
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit in /gcp")
		var datasets []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&datasets)
		return datasets, err
	}

	log.Debugf("Cache miss in /gcp")
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
	sort.Strings(datasets)

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&datasets); err != nil {
		return nil, err
	}
	cli.cache.Set(key, data.Bytes())
	cli.updated = time.Now()
	return datasets, nil
}

func (cli *service) cachedTables(ctx context.Context, d *bigquery.Dataset) ([]string, error) {
	key := cli.proj + "/" + cli.name + "/" + d.DatasetID
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit in /gcp")
		var tables []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&tables)
		return tables, err
	}

	log.Debugf("Cache miss in /gcp")
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
	sort.Strings(tables)

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&tables); err != nil {
		return nil, err
	}
	cli.cache.Set(key, data.Bytes())
	cli.updated = time.Now()
	return tables, nil
}

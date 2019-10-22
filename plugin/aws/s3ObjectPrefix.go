package aws

import (
	"context"

	"github.com/puppetlabs/wash/plugin"

	s3Client "github.com/aws/aws-sdk-go/service/s3"
)

// s3ObjectPrefix represents a common prefix shared by a group of
// S3 objects. Prefixes allow one to view an S3 bucket's contents
// hierarchically. See https://docs.aws.amazon.com/AmazonS3/latest/dev/ListingKeysHierarchy.html
// for more details.
type s3ObjectPrefix struct {
	plugin.EntryBase
	bucket string
	prefix string
	client *s3Client.S3
}

func newS3ObjectPrefix(name string, bucket string, prefix string, client *s3Client.S3) *s3ObjectPrefix {
	objPrefix := &s3ObjectPrefix{
		EntryBase: plugin.NewEntry(name),
	}
	objPrefix.bucket = bucket
	objPrefix.prefix = prefix
	objPrefix.client = client
	return objPrefix
}

func (d *s3ObjectPrefix) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(d, "prefix").SetDescription(s3ObjectPrefixDescription)
}

func (d *s3ObjectPrefix) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&s3ObjectPrefix{}).Schema(),
		(&s3Object{}).Schema(),
	}
}

// List lists all S3 objects and S3 object prefixes that are
// prefixed by the current S3 object prefix
func (d *s3ObjectPrefix) List(ctx context.Context) ([]plugin.Entry, error) {
	return listObjects(ctx, d.client, d.bucket, d.prefix)
}

func (d *s3ObjectPrefix) Delete(ctx context.Context) (bool, error) {
	err := deleteObjects(ctx, d.client, d.bucket, d.prefix)
	return true, err
}

const s3ObjectPrefixDescription = `
This represents a common prefix shared by multiple S3 objects. See the
bucket's description for more details on why we have this kind of entry.
`

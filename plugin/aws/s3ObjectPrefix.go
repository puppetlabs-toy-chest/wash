package aws

import (
	"context"
	"path"

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

func newS3ObjectPrefix(bucket string, prefix string, client *s3Client.S3) *s3ObjectPrefix {
	return &s3ObjectPrefix{
		EntryBase: plugin.NewEntry(path.Base(prefix)),
		bucket:    bucket,
		prefix:    prefix,
		client:    client,
	}
}

// List lists all S3 objects and S3 object prefixes that are
// prefixed by the current S3 object prefix
func (d *s3ObjectPrefix) List(ctx context.Context) ([]plugin.Entry, error) {
	return listObjects(ctx, d.client, d.bucket, d.prefix)
}

package aws

import (
	"context"
	"fmt"
	"path"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	s3Client "github.com/aws/aws-sdk-go/service/s3"
)

// listObjects is a helper that lists the objects which start with a specific
// prefix. We don't make it a method of s3Bucket because the helper's also
// used by s3ObjectPrefix. While we could pass-around the same s3Bucket object to all
// its s3ObjectPrefix children, doing so is not a good idea because (1) it is a bit overkill
// to pass-around an entire object just to access only one of its methods and (2),
// it makes it difficult to refresh the shared s3Bucket object when the original object
// is evicted from the cache.
func listObjects(ctx context.Context, client *s3Client.S3, bucket string, prefix string) ([]plugin.Entry, error) {
	// TODO: Figure out what to log into the journal

	request := &s3Client.ListObjectsInput{
		Bucket:    awsSDK.String(bucket),
		Prefix:    awsSDK.String(prefix),
		Delimiter: awsSDK.String("/"),
	}

	resp, err := client.ListObjectsWithContext(ctx, request)
	if err != nil {
		return nil, err
	}

	// CommonPrefixes represent S3 object prefixes; Contents represent S3 objects.
	numPrefixes := len(resp.CommonPrefixes)
	numObjects := len(resp.Contents)
	entries := make([]plugin.Entry, numPrefixes+numObjects)

	journal.Record(
		ctx,
		"(Bucket %v, Prefix %v): Retrieved %v prefixes and %v objects",
		bucket,
		prefix,
		numPrefixes,
		numObjects,
	)

	for i, p := range resp.CommonPrefixes {
		prefix := awsSDK.StringValue(p.Prefix)
		entries[i] = newS3ObjectPrefix(bucket, prefix, client)
	}

	for i, o := range resp.Contents {
		request := &s3Client.HeadObjectInput{
			Bucket: awsSDK.String(bucket),
			Key:    o.Key,
		}

		key := awsSDK.StringValue(o.Key)
		name := path.Base(key)

		// TODO: Once https://github.com/puppetlabs/wash/issues/123
		// is resolved, we should move the HeadObject calls over to
		// s3Object and cache its response.
		resp, err := client.HeadObjectWithContext(ctx, request)
		if err != nil {
			// TODO: Should we log a warning here instead?
			return nil, fmt.Errorf("could not get the metadata + attributes for object %v: %v", name, err)
		}

		attr := plugin.Attributes{
			Mtime: awsSDK.TimeValue(resp.LastModified),
			// TODO: Check for a negative size
			Size: uint64(awsSDK.Int64Value(resp.ContentLength)),
		}

		// TODO: Right now, resp.Metadata includes the user-specified
		// metadata. What else would be useful to include here?
		//
		// NOTE: Here's everything returned by HeadObjectOutput:
		//
		// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#HeadObjectOutput
		//
		metadata := plugin.ToMetadata(resp.Metadata)

		entries[numPrefixes+i] = newS3Object(attr, metadata, bucket, key, client)
	}

	return entries, nil
}

// s3Bucket represents an S3 bucket.
type s3Bucket struct {
	plugin.EntryBase
	attr   plugin.Attributes
	client *s3Client.S3
}

func newS3Bucket(name string, attr plugin.Attributes, client *s3Client.S3) *s3Bucket {
	return &s3Bucket{
		EntryBase: plugin.NewEntry(name),
		attr:      attr,
		client:    client,
	}
}

func (b *s3Bucket) List(ctx context.Context) ([]plugin.Entry, error) {
	return listObjects(ctx, b.client, b.Name(), "")
}

func (b *s3Bucket) Attr() plugin.Attributes {
	return b.attr
}

// TODO: Implement Metadata. What would be useful information that we could
// include here?

package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"

	"github.com/aws/aws-sdk-go/aws"
	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
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
	// TODO: Clarify this a bit more later. For now, this should be enough.
	//
	// Everything's an object in S3. There is no such thing as a "hierarchy", meaning
	// that hierarchical concepts like directories and files do not exist. However, we
	// can represent S3 objects as a hierarchy using the API's "Prefix" and "Delimiter"
	// options. When these options are set to <prefix> and "/", respectively, the response
	// will contain two values:
	//     * "CommonPrefixes". This is an array of common prefixes where a "CommonPrefix"
	//     represents the group of keys that begin with "CommonPrefix". Let <key> be one of
	//     these keys. Then we can describe "CommonPrefix" as (using pseudocode for clarity):
	//         CommonPrefix = <prefix> + <key>.NonEmptySubstrAfter(<prefix>).UpToAndIncluding("/")
	//
	//     * "Contents". This contains all of the S3 objects whose keys begin with <prefix>,
	//     but do not contain the delimiter "/" after <prefix>. NOTE: These semantics imply that
	//     if an object has <prefix> as its key, then it will also be included in "Contents"
	//     (its key begins with <prefix>, but does not contain "/" after <prefix>).
	//
	// "CommonPrefixes" and "Contents" are mutually exclusive, meaning that an S3 object whose
	// key is grouped by a "CommonPrefix" will not appear in "Contents".
	//
	// As an example, assume our S3 bucket contains the following keys (each key's quoted for
	// clarity)
	//     "/"
	//     "//"
	//     "/foo"
	//     "foo/bar"
	//     "foo/bar/baz"
	//     "bar"
	//     "baz"
	//
	// Then if <prefix> == "", our response will consist of:
	//     resp.CommonPrefixes = ["/", "foo/"]
	//     resp.Contents = ["bar", "baz"]
	//
	// This should make sense. The "/", "//", and "/foo" keys are grouped by the
	// "/" common prefix, while the "foo/bar", "foo/bar/baz" keys are grouped by the
	// "foo/" common prefix. "bar" and "baz" begin with "", but do not contain the delimiter
	// "/" after "".
	//
	// Let's check that "/" and "foo/" are common prefixes. Starting with "/", we can use
	// the "/foo" key to see that:
	//     CommonPrefix = <prefix> + <key>.NonEmptySubstrAfter(<prefix>).UpToAndIncluding("/")
	//     CommonPrefix = "" + "/foo".NonEmptySubstrAfter("").UpToAndIncluding("/")
	//     CommonPrefix = "/foo".UpToAndIncluding("/")
	//     CommonPrefix = "/"
	//
	// which is correct. Similarly for "foo/", we can use the "foo/bar/baz" key to see that:
	//     CommonPrefix = <prefix> + <key>.NonEmptySubstrAfter(<prefix>).UpToAndIncluding("/")
	//     CommonPrefix = "" + "foo/bar/baz".NonEmptySubstrAfter("").UpToAndIncluding("/")
	//     CommonPrefix = "foo/bar/baz".UpToAndIncluding("/")
	//     CommonPrefix = "foo/"
	//
	// which is also correct.  Now what happens if we pass-in "/" as our prefix? Then the response
	// will consist of:
	//     resp.CommonPrefixes = ["//"]
	//     resp.Contents = ["/foo"]
	//
	// This should also make sense. We see that the "//" key is grouped by the "//" common prefix
	// (use the CommonPrefix "equation" with <prefix> = "/" to check this). Similarly, the "/foo"
	// key begins with "/", but is not grouped by a common prefix because it does not contain the
	// delimiter "/" after the prefix "/".
	//
	// Finally, what happens if we pass-in "foo/" as our prefix? Then the response consists of
	//     resp.CommonPrefixes = ["foo/bar/"]
	//     resp.Contents = ["foo/bar"]
	//
	// This should also make sense. The "foo/bar/baz" key is grouped by the "foo/bar/" common
	// prefix. Similarly, the "foo/bar" key begins with "foo/", but does not contain the delimiter
	// "/" after the prefix "foo/".
	//
	// Thus, we can view S3 objects hierarchically by making the "CommonPrefixes" our directories,
	// and the "Contents" our files. These are modeled by the "s3ObjectPrefix" and "s3Object" classes,
	// respectively.
	request := &s3Client.ListObjectsInput{
		Bucket:    awsSDK.String(bucket),
		Prefix:    awsSDK.String(prefix),
		Delimiter: awsSDK.String("/"),
	}
	resp, err := client.ListObjectsWithContext(ctx, request)
	if err != nil {
		return nil, err
	}
	numPrefixes := len(resp.CommonPrefixes)
	numObjects := len(resp.Contents)
	entries := make([]plugin.Entry, 0, numPrefixes+numObjects)

	journal.Record(
		ctx,
		"(Bucket %v, Prefix %v): Retrieved %v prefixes and %v objects",
		bucket,
		prefix,
		numPrefixes,
		numObjects,
	)

	// resp.CommonPrefixes represents all of the object keys
	for _, p := range resp.CommonPrefixes {
		commonPrefix := awsSDK.StringValue(p.Prefix)
		name := strings.TrimPrefix(commonPrefix, prefix)
		if name != "/" {
			name = strings.TrimSuffix(name, "/")
		}

		entries = append(entries, newS3ObjectPrefix(name, bucket, commonPrefix, client))
	}

	for _, o := range resp.Contents {
		key := awsSDK.StringValue(o.Key)
		name := strings.TrimPrefix(key, prefix)
		if name == "" {
			// key == <prefix> so skip it. This is what the AWS console does.
			continue
		}
		entries = append(entries, newS3Object(o, name, bucket, key, client))
	}

	return entries, nil
}

// s3Bucket represents an S3 bucket.
type s3Bucket struct {
	plugin.EntryBase
	ctime   time.Time
	client  *s3Client.S3
	session *session.Session
}

func newS3Bucket(name string, ctime time.Time, session *session.Session) *s3Bucket {
	bucket := &s3Bucket{
		EntryBase: plugin.NewEntry(name),
		ctime:     ctime,
		client:    s3Client.New(session),
		session:   session,
	}

	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(bucket.ctime).
		SetMtime(bucket.ctime).
		SetAtime(bucket.ctime)
	bucket.SetInitialAttributes(attr)

	return bucket
}

func (b *s3Bucket) List(ctx context.Context) ([]plugin.Entry, error) {
	if _, err := b.getRegion(ctx); err != nil {
		return nil, err
	}
	return listObjects(ctx, b.client, b.Name(), "")
}

func (b *s3Bucket) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	request := &s3Client.GetBucketTaggingInput{
		Bucket: awsSDK.String(b.Name()),
	}

	resp, err := b.client.GetBucketTaggingWithContext(ctx, request)

	var metadata plugin.EntryMetadata
	if err == nil {
		metadata = plugin.ToMeta(resp)
	} else if awserr, ok := err.(awserr.Error); ok {
		// Check if this is a NoSuchTagSet error. If yes, then that means
		// this bucket doesn't have any tags.
		//
		// NOTE: See https://github.com/boto/boto3/issues/341#issuecomment-186007537
		// if you're interested in knowing why AWS does not return
		// an empty TagSet instead of a NoSuchTagSet error
		if awserr.Code() == "NoSuchTagSet" {
			metadata = plugin.EntryMetadata{}
		} else {
			return nil, err
		}
	} else {
		// We have a non-AWS related error
		return nil, err
	}

	region, err := b.getRegion(ctx)
	if err != nil {
		return nil, err
	}
	metadata["region"] = region
	metadata["ctime"] = b.ctime

	return metadata, nil
}

func (b *s3Bucket) getRegion(ctx context.Context) (string, error) {
	// Note that the callback to CachedOp also creates a new client for that region.
	// We use CachedOp with a long expiration to ensure region is fetched infrequently.
	// You can force a retry by deleting the cache entry if there was an error.
	resp, err := plugin.CachedOp("Region", b, 24*time.Hour, func() (interface{}, error) {
		locRequest := &s3Client.GetBucketLocationInput{Bucket: awsSDK.String(b.Name())}
		resp, err := b.client.GetBucketLocationWithContext(ctx, locRequest)
		if err != nil {
			return nil, fmt.Errorf("could not get the region of bucket %v: %v", b.Name(), err)
		}

		// The response will be empty if the bucket is in Amazon's default region (us-east-1)
		region := s3Client.NormalizeBucketLocation(awsSDK.StringValue(resp.LocationConstraint))
		// Update client to be region-specific
		b.client = s3Client.New(b.session, aws.NewConfig().WithRegion(region))

		return region, nil
	})

	if err != nil {
		return "", err
	}

	return resp.(string), nil
}

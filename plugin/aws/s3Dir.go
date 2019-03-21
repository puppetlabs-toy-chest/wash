package aws

import (
	"context"
	"fmt"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	s3Client "github.com/aws/aws-sdk-go/service/s3"
)

// s3Dir represents the resources/s3 directory
type s3Dir struct {
	plugin.EntryBase
	session *session.Session
	client  *s3Client.S3
}

func newS3Dir(session *session.Session) *s3Dir {
	return &s3Dir{
		EntryBase: plugin.NewEntry("s3"),
		session:   session,
		client:    s3Client.New(session),
	}
}

// List lists the buckets.
func (s *s3Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	resp, err := s.client.ListBucketsWithContext(ctx, nil)
	if err != nil {
		return nil, err
	}

	journal.Record(ctx, "Listing %v S3 buckets", len(resp.Buckets))

	buckets := make([]plugin.Entry, len(resp.Buckets))
	for i, bucket := range resp.Buckets {
		name := bucket.Name

		locRequest := &s3Client.GetBucketLocationInput{
			Bucket: name,
		}
		resp, err := s.client.GetBucketLocationWithContext(ctx, locRequest)
		if err != nil {
			// TODO: Should we log a warning and continue instead of returning an
			// error?
			return nil, fmt.Errorf("could not get the region of bucket %v: %v", name, err)
		}

		// The response will be empty if the bucket is in Amazon's default region (us-east-1)
		region := "us-east-1"
		if resp.LocationConstraint != nil {
			region = awsSDK.StringValue(resp.LocationConstraint)
		}

		cfg := awsSDK.NewConfig()
		cfg.WithRegion(region)

		buckets[i] = newS3Bucket(
			awsSDK.StringValue(name),
			awsSDK.TimeValue(bucket.CreationDate),
			region,
			s3Client.New(s.session, cfg),
		)
	}

	return buckets, nil
}

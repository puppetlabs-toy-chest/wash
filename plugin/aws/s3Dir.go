package aws

import (
	"context"

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

func newS3Dir(parent plugin.Entry, session *session.Session) *s3Dir {
	return &s3Dir{
		EntryBase: parent.NewEntry("s3"),
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
		buckets[i] = newS3Bucket(
			s,
			awsSDK.StringValue(bucket.Name),
			awsSDK.TimeValue(bucket.CreationDate),
			s.session,
		)
	}

	return buckets, nil
}

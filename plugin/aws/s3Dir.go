package aws

import (
	"context"
	"fmt"

	"github.com/puppetlabs/wash/activity"
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

func newS3Dir(ctx context.Context, session *session.Session) *s3Dir {
	s3Dir := &s3Dir{
		EntryBase: plugin.NewEntry("s3"),
	}
	s3Dir.session = session
	s3Dir.client = s3Client.New(session)
	if _, err := plugin.List(ctx, s3Dir); err != nil {
		s3Dir.MarkInaccessible(ctx, err)
	}
	return s3Dir
}

func (s *s3Dir) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(s, "s3").IsSingleton()
}

func (s *s3Dir) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&s3Bucket{}).Schema(),
	}
}

// List lists the buckets.
func (s *s3Dir) List(ctx context.Context) ([]plugin.Entry, error) {
	resp, err := s.client.ListBucketsWithContext(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Error listing buckets: %w", err)
	}

	activity.Record(ctx, "Listing %v S3 buckets", len(resp.Buckets))

	buckets := make([]plugin.Entry, len(resp.Buckets))
	for i, bucket := range resp.Buckets {
		buckets[i] = newS3Bucket(
			awsSDK.StringValue(bucket.Name),
			awsSDK.TimeValue(bucket.CreationDate),
			s.session,
		)
	}

	return buckets, nil
}

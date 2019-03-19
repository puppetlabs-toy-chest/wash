package aws

import (
	"context"
	"errors"
	"io"
	"path"
	"strconv"

	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	s3Client "github.com/aws/aws-sdk-go/service/s3"
)

// s3Object represents an S3 object.
type s3Object struct {
	plugin.EntryBase
	attr     plugin.Attributes
	metadata plugin.MetadataMap
	bucket   string
	key      string
	client   *s3Client.S3
}

func newS3Object(attr plugin.Attributes, metadata plugin.MetadataMap, bucket string, key string, client *s3Client.S3) *s3Object {
	o := &s3Object{
		EntryBase: plugin.NewEntry(path.Base(key)),
		attr:      attr,
		metadata:  metadata,
		bucket:    bucket,
		key:       key,
		client:    client,
	}
	o.TurnOffCachingFor(plugin.Metadata)

	return o
}

func (o *s3Object) Attr() plugin.Attributes {
	return o.attr
}

func (o *s3Object) Metadata(ctx context.Context) (plugin.MetadataMap, error) {
	return o.metadata, nil
}

func (o *s3Object) fetchContent(off int64) (io.ReadCloser, error) {
	request := &s3Client.GetObjectInput{
		Bucket: awsSDK.String(o.bucket),
		Key:    awsSDK.String(o.key),
		Range:  awsSDK.String("bytes=" + strconv.FormatInt(off, 10) + "-"),
	}

	resp, err := o.client.GetObject(request)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (o *s3Object) Open(ctx context.Context) (plugin.SizedReader, error) {
	return &s3ObjectReader{o}, nil
}

func (o *s3Object) Stream(context.Context) (io.Reader, error) {
	return o.fetchContent(0)
}

// TODO: Optimize this class later. For now, the simple implementation is
// enough to get `cat` working for small objects and, for large objects, enough
// to get it to print something to stdout without having to wait for the entire
// object to be downloaded.
//
// https://github.com/kahing/goofys/blob/master/internal/file.go has some prior
// art we could use to optimize this.
type s3ObjectReader struct {
	o *s3Object
}

func (s *s3ObjectReader) closeContent(content io.ReadCloser) {
	if err := content.Close(); err != nil {
		journal.Record(context.Background(), "aws.s3ObjectReader.ReadAt: failed to close %v's content: %v", s.o.key, err)
	}
}

func (s *s3ObjectReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("aws.s3ObjectReader.ReadAt: negative offset")
	}

	if off >= s.Size() {
		return 0, io.EOF
	}

	content, err := s.o.fetchContent(off)
	if err != nil {
		return 0, err
	}
	defer s.closeContent(content)

	return io.ReadFull(content, p)
}

func (s *s3ObjectReader) Size() int64 {
	return int64(s.o.Attr().Size)
}

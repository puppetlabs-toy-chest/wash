package aws

import (
	"context"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"

	awsSDK "github.com/aws/aws-sdk-go/aws"
	s3Client "github.com/aws/aws-sdk-go/service/s3"
)

// s3Object represents an S3 object.
type s3Object struct {
	plugin.EntryBase
	bucket string
	key    string
	client *s3Client.S3
}

func newS3Object(o *s3Client.Object, name string, bucket string, key string, client *s3Client.S3) *s3Object {
	s3Obj := &s3Object{
		EntryBase: plugin.NewEntry(name),
		bucket:    bucket,
		key:       key,
		client:    client,
	}
	s3Obj.DisableCachingFor(plugin.MetadataOp)

	// S3 objects do not have a "creation time"; they're treated as atomic
	// blobs that get replaced whenever the user uploads new data. Thus, we
	// use the mtime as its creation date. See https://stackoverflow.com/questions/27746760/how-do-i-get-the-s3-keys-created-date-with-boto
	// for more details.
	//
	// TODO: Export a mungeSize helper to abstract away the common
	// logic of validating a negative size
	mtime := awsSDK.TimeValue(o.LastModified)
	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(mtime).
		SetMtime(mtime).
		SetAtime(mtime).
		SetSize(uint64(awsSDK.Int64Value(o.Size)))
	s3Obj.SetAttributes(attr)

	return s3Obj
}

func (o *s3Object) cachedHeadObject(ctx context.Context) (*s3Client.HeadObjectOutput, error) {
	resp, err := plugin.CachedOp(ctx, "HeadObject", o, 15*time.Second, func() (interface{}, error) {
		request := &s3Client.HeadObjectInput{
			Bucket: awsSDK.String(o.bucket),
			Key:    awsSDK.String(o.key),
		}

		return o.client.HeadObjectWithContext(ctx, request)
	})

	if err != nil {
		return nil, err
	}

	return resp.(*s3Client.HeadObjectOutput), nil
}

func (o *s3Object) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	metadata, err := o.cachedHeadObject(ctx)
	if err != nil {
		return nil, nil
	}

	return plugin.ToMeta(metadata), nil
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
	return &s3ObjectReader{o: o}, nil
}

func (o *s3Object) Stream(context.Context) (io.ReadCloser, error) {
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
		activity.Record(context.Background(), "aws.s3ObjectReader.ReadAt: failed to close %v's content: %v", s.o.key, err)
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
	attr := plugin.Attributes(s.o)
	return int64(attr.Size())
}

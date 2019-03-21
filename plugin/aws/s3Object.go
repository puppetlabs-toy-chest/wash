package aws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"time"

	"github.com/puppetlabs/wash/journal"
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

func newS3Object(bucket string, key string, client *s3Client.S3) *s3Object {
	o := &s3Object{
		EntryBase: plugin.NewEntry(path.Base(key)),
		bucket:    bucket,
		key:       key,
		client:    client,
	}
	o.DisableCachingFor(plugin.Metadata)

	return o
}

type headObjectResult struct {
	attr     plugin.Attributes
	metadata *s3Client.HeadObjectOutput
}

func (o *s3Object) cachedHeadObject(ctx context.Context) (headObjectResult, error) {
	resp, err := plugin.CachedOp("HeadObject", o, 15*time.Second, func() (interface{}, error) {
		request := &s3Client.HeadObjectInput{
			Bucket: awsSDK.String(o.bucket),
			Key:    awsSDK.String(o.key),
		}

		resp, err := o.client.HeadObjectWithContext(ctx, request)
		if err != nil {
			return nil, err
		}
		result := headObjectResult{
			metadata: resp,
		}

		size := awsSDK.Int64Value(result.metadata.ContentLength)
		if size < 0 {
			err := fmt.Errorf("got a negative value of %v for the size of the %v object's content", size, o.key)
			return plugin.Attributes{}, err
		}

		o.Ctime = awsSDK.TimeValue(result.metadata.LastModified)
		result.attr, _ = o.EntryBase.Attr(ctx)
		result.attr.Size = uint64(size)

		return result, nil
	})

	if err != nil {
		return headObjectResult{}, err
	}

	return resp.(headObjectResult), nil
}

func (o *s3Object) Attr(ctx context.Context) (plugin.Attributes, error) {
	result, err := o.cachedHeadObject(ctx)
	if err != nil {
		return plugin.Attributes{}, nil
	}

	return result.attr, nil
}

func (o *s3Object) Metadata(ctx context.Context) (plugin.MetadataMap, error) {
	result, err := o.cachedHeadObject(ctx)
	if err != nil {
		return plugin.MetadataMap{}, nil
	}

	return plugin.ToMetadata(result.metadata), nil
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
	attr, err := o.Attr(ctx)
	if err != nil {
		return nil, err
	}

	return &s3ObjectReader{o: o, size: int64(attr.Size)}, nil
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
	// See the comments in s3ObjectReader#Size to understand how this field's
	// used.
	size int64
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
	ctx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFunc()

	// We'd like to return s.o.Attr().Size here. Unfortunately since
	// s.o.Attr() is calculated via. an API request, there's a chance
	// that it could error. If that happens, we return s.size instead
	// as a fallback. We could change Size()'s return signature to
	// (int64, error), but there's no good reason to do that right now.
	attr, err := s.o.Attr(ctx)
	if err != nil {
		return s.size
	}

	return int64(attr.Size)
}

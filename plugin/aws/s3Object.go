package aws

import (
	"bytes"
	"context"
	"io/ioutil"
	"strconv"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"

	"github.com/aws/aws-sdk-go/aws"
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
	}
	s3Obj.bucket = bucket
	s3Obj.key = key
	s3Obj.client = client

	// S3 objects do not have a "creation time"; they're treated as atomic
	// blobs that get replaced whenever the user uploads new data. Thus, we
	// use the mtime as its creation date. See https://stackoverflow.com/questions/27746760/how-do-i-get-the-s3-keys-created-date-with-boto
	// for more details.
	//
	// TODO: Export a mungeSize helper to abstract away the common
	// logic of validating a negative size
	mtime := awsSDK.TimeValue(o.LastModified)
	s3Obj.
		Attributes().
		SetCrtime(mtime).
		SetMtime(mtime).
		SetCtime(mtime).
		SetAtime(mtime).
		SetSize(uint64(awsSDK.Int64Value(o.Size))).
		SetMeta(o)

	return s3Obj
}

func (o *s3Object) Schema() *plugin.EntrySchema {
	return plugin.
		NewEntrySchema(o, "object").
		SetDescription(s3ObjectDescription).
		SetMetaAttributeSchema(s3Client.Object{}).
		SetMetadataSchema(s3Client.HeadObjectOutput{})
}

func (o *s3Object) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	request := &s3Client.HeadObjectInput{
		Bucket: awsSDK.String(o.bucket),
		Key:    awsSDK.String(o.key),
	}

	metadata, err := o.client.HeadObjectWithContext(ctx, request)
	if err != nil {
		return nil, err
	}

	return plugin.ToJSONObject(metadata), nil
}

func (o *s3Object) Read(ctx context.Context, size int64, offset int64) ([]byte, error) {
	request := &s3Client.GetObjectInput{
		Bucket: awsSDK.String(o.bucket),
		Key:    awsSDK.String(o.key),
		Range:  awsSDK.String("bytes=" + strconv.FormatInt(offset, 10) + "-" + strconv.FormatInt(offset+size, 10)),
	}

	resp, err := o.client.GetObjectWithContext(ctx, request)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			activity.Record(ctx, "Error closing S3 GetObject response body: %v", err)
		}
	}()

	activity.Record(ctx, "S3 object read response: %+v", *resp)
	return ioutil.ReadAll(resp.Body)
}

func (o *s3Object) Write(ctx context.Context, p []byte) error {
	request := &s3Client.PutObjectInput{
		Bucket: awsSDK.String(o.bucket),
		Key:    awsSDK.String(o.key),
		Body:   bytes.NewReader(p),
	}

	resp, err := o.client.PutObjectWithContext(ctx, request)
	if err != nil {
		return err
	}

	activity.Record(ctx, "S3 object write response: %+v", *resp)
	return nil
}

func (o *s3Object) Delete(ctx context.Context) (bool, error) {
	_, err := o.client.DeleteObjectWithContext(ctx, &s3Client.DeleteObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(o.key),
	})
	return true, err
}

const s3ObjectDescription = `
This is an S3 object. See the bucket's description for more details on
why we have this kind of entry.
`

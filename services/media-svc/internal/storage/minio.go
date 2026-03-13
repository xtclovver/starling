package storage

import (
	"bytes"
	"context"
	"time"

	"github.com/minio/minio-go/v7"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(client *minio.Client, bucket string) *MinIOClient {
	return &MinIOClient{client: client, bucket: bucket}
}

func (c *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (c *MinIOClient) Upload(ctx context.Context, objectKey string, data []byte, contentType string) error {
	_, err := c.client.PutObject(ctx, c.bucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (c *MinIOClient) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := c.client.PresignedGetObject(ctx, c.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (c *MinIOClient) GetPresignedUploadURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := c.client.PresignedPutObject(ctx, c.bucket, objectKey, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (c *MinIOClient) Delete(ctx context.Context, objectKey string) error {
	return c.client.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{})
}

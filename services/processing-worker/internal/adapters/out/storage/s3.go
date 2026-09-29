package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client *minio.Client
	bucket string
	region string
}

func NewS3Storage(endpoint, accessKey, secretKey, bucket, region string) (*S3Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
		Region: region,
	})
	if err != nil {
		return nil, err
	}

	s := &S3Storage{client: client, bucket: bucket, region: region}

	ctx := context.Background()
	var lastErr error
	for i := 0; i < 30; i++ {
		if err := s.ensureBucket(ctx); err == nil {
			return s, nil
		} else {
			lastErr = err
		}
		time.Sleep(time.Second)
	}
	return nil, lastErr
}

func (s *S3Storage) ensureBucket(ctx context.Context) error {
	ok, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region})
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *S3Storage) Upload(ctx context.Context, key string, data io.Reader) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, data, -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

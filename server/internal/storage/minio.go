package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	UseSSL          bool
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Endpoint) != "" ||
		strings.TrimSpace(c.AccessKeyID) != "" ||
		strings.TrimSpace(c.SecretAccessKey) != "" ||
		strings.TrimSpace(c.Bucket) != "" ||
		strings.TrimSpace(c.Region) != ""
}

func (c Config) Validate() error {
	missing := make([]string, 0, 4)

	if strings.TrimSpace(c.Endpoint) == "" {
		missing = append(missing, "MINIO_ENDPOINT")
	}
	if strings.TrimSpace(c.AccessKeyID) == "" {
		missing = append(missing, "MINIO_ACCESS_KEY")
	}
	if strings.TrimSpace(c.SecretAccessKey) == "" {
		missing = append(missing, "MINIO_SECRET_KEY")
	}
	if strings.TrimSpace(c.Bucket) == "" {
		missing = append(missing, "MINIO_BUCKET")
	}

	if len(missing) > 0 {
		return fmt.Errorf("incomplete MinIO configuration; missing %s", strings.Join(missing, ", "))
	}

	return nil
}

type Client struct {
	client *minio.Client
	bucket string
}

func NewMinIO(ctx context.Context, config Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Region: config.Region,
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}

	exists, err := minioClient.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", config.Bucket, err)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{Region: config.Region}); err != nil {
			exists, bucketErr := minioClient.BucketExists(ctx, config.Bucket)
			if bucketErr != nil {
				return nil, fmt.Errorf("create bucket %q: %w", config.Bucket, err)
			}
			if !exists {
				return nil, fmt.Errorf("create bucket %q: %w", config.Bucket, err)
			}
		}
	}

	return &Client{
		client: minioClient,
		bucket: config.Bucket,
	}, nil
}

func (c *Client) Bucket() string {
	if c == nil {
		return ""
	}

	return c.bucket
}

func (c *Client) MinIO() *minio.Client {
	if c == nil {
		return nil
	}

	return c.client
}

package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

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
	return nil, nil

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

func (c *Client) PutObject(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	if c == nil || c.client == nil {
		return minio.UploadInfo{}, fmt.Errorf("MinIO client is not configured")
	}

	return c.client.PutObject(ctx, c.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
}

func (c *Client) GetObject(ctx context.Context, objectKey string) (*minio.Object, minio.ObjectInfo, error) {
	if c == nil || c.client == nil {
		return nil, minio.ObjectInfo{}, fmt.Errorf("MinIO client is not configured")
	}

	object, err := c.client.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}

	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, minio.ObjectInfo{}, err
	}

	return object, info, nil
}

func (c *Client) RemoveObject(ctx context.Context, objectKey string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("MinIO client is not configured")
	}

	return c.client.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{})
}

func TicketAttachmentObjectKey(_ string, attachmentID string, fileName string) string {
	base := strings.TrimSpace(attachmentID)
	ext := sanitizeObjectKeyExtension(fileName)
	if ext == "" {
		return base
	}

	return base + "." + ext
}

func sanitizeObjectKeyExtension(fileName string) string {
	trimmed := strings.TrimSpace(fileName)
	dotIndex := strings.LastIndex(trimmed, ".")
	if dotIndex < 0 || dotIndex == len(trimmed)-1 {
		return ""
	}

	var builder strings.Builder
	for _, char := range strings.ToLower(trimmed[dotIndex+1:]) {
		if unicode.IsDigit(char) || (char >= 'a' && char <= 'z') {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

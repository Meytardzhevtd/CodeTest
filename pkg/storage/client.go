package storage

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc             *minio.Client
	bucket         string
	publicEndpoint string
	useSSL         bool
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool

	PublicEndpoint string
}

func NewClient(cfg Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: create minio client: %w", err)
	}

	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}

	return &Client{
		mc:             mc,
		bucket:         cfg.Bucket,
		publicEndpoint: publicEndpoint,
		useSSL:         cfg.UseSSL,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("storage: check bucket exists: %w", err)
	}
	if exists {
		return nil
	}

	if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("storage: create bucket %q: %w", c.bucket, err)
	}
	return nil
}

func (c *Client) EnsurePublicBucket(ctx context.Context) error {
	if err := c.EnsureBucket(ctx); err != nil {
		return err
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, c.bucket)

	if err := c.mc.SetBucketPolicy(ctx, c.bucket, policy); err != nil {
		return fmt.Errorf("storage: set public read policy on bucket %q: %w", c.bucket, err)
	}
	return nil
}

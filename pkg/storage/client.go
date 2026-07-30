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
	Endpoint  string // например "localhost:9000" или "minio:9000" внутри docker-сети, БЕЗ http://
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool

	// PublicEndpoint is the host:port used to build links returned to
	// browsers (e.g. avatar URLs). It can differ from Endpoint: inside
	// docker-compose the server talks to MinIO over the internal network
	// (Endpoint="minio:9000"), but a browser on the host needs the
	// published port (PublicEndpoint="localhost:9001"). Falls back to
	// Endpoint when unset.
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

// EnsurePublicBucket creates the bucket if needed and (idempotently) grants
// anonymous read access to its objects, so uploaded files can be linked to
// directly (e.g. avatars rendered in <img> tags) without proxying through
// the API.
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

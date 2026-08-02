package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

func (c *Client) UploadAvatar(ctx context.Context, userID string, data io.Reader, size int64, contentType string) (string, error) {
	if _, err := c.mc.PutObject(ctx, c.bucket, userID, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return "", fmt.Errorf("storage: upload avatar %q: %w", userID, err)
	}
	return userID, nil
}

func (c *Client) DeleteAvatar(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage: delete avatar %q: %w", key, err)
	}
	return nil
}

func (c *Client) PublicURL(key string) string {
	scheme := "http"
	if c.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, c.publicEndpoint, c.bucket, key)
}

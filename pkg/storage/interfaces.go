package storage

import (
	"context"
	"io"
)

type Reader interface {
	ListTests(ctx context.Context, taskID string) ([]TestCase, error)
	GetTest(ctx context.Context, tc TestCase) (input io.ReadCloser, output io.ReadCloser, err error)
}

type Writer interface {
	UploadTest(ctx context.Context, taskID string, testNum int, input io.Reader, inputSize int64, output io.Reader, outputSize int64) error
	DeleteTask(ctx context.Context, taskID string) error
}

var (
	_ Reader = (*Client)(nil)
	_ Writer = (*Client)(nil)
)

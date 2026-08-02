package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()

	cfg := Config{
		Endpoint:  envOr("MINIO_ENDPOINT", "localhost:9001"),
		AccessKey: envOr("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: envOr("MINIO_SECRET_KEY", "minioadmin123"),
		Bucket:    "codetest-tests-" + uuid.NewString(),
		UseSSL:    false,
	}

	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	if err := c.EnsureBucket(ctx); err != nil {
		t.Skipf("MinIO not reachable at %s, skipping integration test: %v", cfg.Endpoint, err)
	}

	t.Cleanup(func() {
		objCh := c.mc.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{Recursive: true})
		for rmErr := range c.mc.RemoveObjects(ctx, c.bucket, objCh, minio.RemoveObjectsOptions{}) {
			_ = rmErr
		}
		_ = c.mc.RemoveBucket(ctx, c.bucket)
	})

	return c
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustUploadTest(t *testing.T, c *Client, taskID string, num int, in, out string) {
	t.Helper()
	err := c.UploadTest(context.Background(), taskID, num,
		bytes.NewReader([]byte(in)), int64(len(in)),
		bytes.NewReader([]byte(out)), int64(len(out)),
	)
	if err != nil {
		t.Fatalf("UploadTest(%d): %v", num, err)
	}
}

func readAll(t *testing.T, r io.ReadCloser) string {
	t.Helper()
	defer r.Close() //nolint:errcheck,noctx
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

func TestUploadListGetDeleteTest(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	taskID := uuid.NewString()

	mustUploadTest(t, c, taskID, 1, "in-1", "out-1")
	mustUploadTest(t, c, taskID, 2, "in-2", "out-2")

	tests, err := c.ListTests(ctx, taskID)
	if err != nil {
		t.Fatalf("ListTests: %v", err)
	}
	if len(tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(tests))
	}
	if tests[0].Number != 1 || tests[1].Number != 2 {
		t.Fatalf("expected tests ordered [1, 2], got [%d, %d]", tests[0].Number, tests[1].Number)
	}

	in, out, err := c.GetTest(ctx, tests[0])
	if err != nil {
		t.Fatalf("GetTest: %v", err)
	}
	if got := readAll(t, in); got != "in-1" {
		t.Errorf("input = %q, want %q", got, "in-1")
	}
	if got := readAll(t, out); got != "out-1" {
		t.Errorf("output = %q, want %q", got, "out-1")
	}

	if err := c.DeleteTask(ctx, taskID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if _, err := c.ListTests(ctx, taskID); err == nil {
		t.Fatal("expected ListTests to fail after DeleteTask, got nil error")
	}
}

func TestListTests_NumberBeyondPaddingWidth(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	taskID := uuid.NewString()

	mustUploadTest(t, c, taskID, 1000, "in", "out")

	tests, err := c.ListTests(ctx, taskID)
	if err != nil {
		t.Fatalf("ListTests: %v", err)
	}
	if len(tests) != 1 || tests[0].Number != 1000 {
		t.Fatalf("expected test 1000 to be listed, got %+v", tests)
	}
}

func TestListTests_IncompleteTestErrors(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	taskID := uuid.NewString()

	if _, err := c.mc.PutObject(ctx, c.bucket, testKey(taskID, 1, "in"),
		bytes.NewReader([]byte("in")), 2, minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		t.Fatalf("seed PutObject: %v", err)
	}

	if _, err := c.ListTests(ctx, taskID); err == nil {
		t.Fatal("expected ListTests to error on an incomplete test pair, got nil")
	}
}

func TestListTests_NoTests(t *testing.T) {
	c := newTestClient(t)
	if _, err := c.ListTests(context.Background(), uuid.NewString()); err == nil {
		t.Fatal("expected error for a task with no tests, got nil")
	}
}

func TestUploadTest_CleansUpOrphanOnPartialFailure(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	taskID := uuid.NewString()

	err := c.UploadTest(ctx, taskID, 1,
		bytes.NewReader([]byte("in-data")), int64(len("in-data")),
		bytes.NewReader([]byte("out")), 999, // declared size doesn't match actual data
	)
	if err == nil {
		t.Fatal("expected UploadTest to fail on output size mismatch")
	}

	_, statErr := c.mc.StatObject(ctx, c.bucket, testKey(taskID, 1, "in"), minio.StatObjectOptions{})
	if statErr == nil {
		t.Fatal("expected orphaned .in object to be cleaned up, but it still exists")
	}
}

func TestGetTest_MissingObjectErrorsImmediately(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	taskID := uuid.NewString()

	tc := TestCase{
		Number: 1,
		InKey:  testKey(taskID, 1, "in"),
		OutKey: testKey(taskID, 1, "out"),
	}

	_, _, err := c.GetTest(ctx, tc)
	if err == nil {
		t.Fatal("expected GetTest to error immediately for a missing object, got nil")
	}
}

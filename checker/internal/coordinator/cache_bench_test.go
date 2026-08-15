package coordinator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
	"github.com/meytardzhevtd/CodeTest/pkg/storage"
)

const benchPayloadBytes = 1024

var benchTestCounts = []int{1, 10, 50, 100}

var benchSink []*judgepb.TestCase

func benchEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newBenchStore(b *testing.B) *storage.Client {
	b.Helper()

	endpoint := benchEnvOr("MINIO_ENDPOINT", "localhost:9001")
	accessKey := benchEnvOr("MINIO_ACCESS_KEY", "minioadmin")
	secretKey := benchEnvOr("MINIO_SECRET_KEY", "minioadmin123")
	bucket := "codetest-bench-" + uuid.NewString()

	client, err := storage.NewClient(storage.Config{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		UseSSL:    false,
	})
	if err != nil {
		b.Fatalf("new storage client: %v", err)
	}

	if err := client.EnsureBucket(context.Background()); err != nil {
		b.Skipf("MinIO not reachable at %s, skipping benchmark: %v", endpoint, err)
	}

	raw, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		b.Fatalf("new minio client: %v", err)
	}

	b.Cleanup(func() {
		if err := raw.RemoveBucket(context.Background(), bucket); err != nil {
			b.Logf("remove bucket %s: %v", bucket, err)
		}
	})

	return client
}

func newBenchCache(b *testing.B) (*TestCache, *redis.Client) {
	b.Helper()

	addr := benchEnvOr("REDIS_ADDR", "localhost:6380")
	client := redis.NewClient(&redis.Options{Addr: addr})

	if err := client.Ping(context.Background()).Err(); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			b.Logf("close redis: %v", closeErr)
		}
		b.Skipf("Redis not reachable at %s, skipping benchmark: %v", addr, err)
	}

	b.Cleanup(func() {
		if err := client.Close(); err != nil {
			b.Logf("close redis: %v", err)
		}
	})

	return NewTestCache(client), client
}

func seedTask(b *testing.B, store *storage.Client, tests int) string {
	b.Helper()

	taskID := uuid.NewString()
	input := strings.Repeat("i", benchPayloadBytes)
	output := strings.Repeat("o", benchPayloadBytes)
	ctx := context.Background()

	for n := 1; n <= tests; n++ {
		err := store.UploadTest(ctx, taskID, n,
			bytes.NewReader([]byte(input)), int64(len(input)),
			bytes.NewReader([]byte(output)), int64(len(output)),
		)
		if err != nil {
			b.Fatalf("upload test %d: %v", n, err)
		}
	}

	b.Cleanup(func() {
		if err := store.DeleteTask(context.Background(), taskID); err != nil {
			b.Logf("delete task %s: %v", taskID, err)
		}
	})

	return taskID
}

func BenchmarkLoadTestCases_MinIO(b *testing.B) {
	store := newBenchStore(b)
	ctx := context.Background()

	for _, count := range benchTestCounts {
		taskID := seedTask(b, store, count)

		b.Run(fmt.Sprintf("N=%d", count), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cases, err := loadTestCases(ctx, store, taskID)
				if err != nil {
					b.Fatalf("load test cases: %v", err)
				}
				benchSink = cases
			}
		})
	}
}

func BenchmarkTestCache_Get_Redis(b *testing.B) {
	store := newBenchStore(b)
	cache, rdb := newBenchCache(b)
	ctx := context.Background()

	for _, count := range benchTestCounts {
		taskID := seedTask(b, store, count)

		cases, err := loadTestCases(ctx, store, taskID)
		if err != nil {
			b.Fatalf("load test cases: %v", err)
		}
		cache.Set(ctx, taskID, cases)

		key := testCacheKeyPrefix + taskID
		b.Cleanup(func() {
			if err := rdb.Del(context.Background(), key).Err(); err != nil {
				b.Logf("delete redis key %s: %v", key, err)
			}
		})

		b.Run(fmt.Sprintf("N=%d", count), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cached, ok := cache.Get(ctx, taskID)
				if !ok {
					b.Fatalf("unexpected cache miss for task %s", taskID)
				}
				benchSink = cached
			}
		})
	}
}

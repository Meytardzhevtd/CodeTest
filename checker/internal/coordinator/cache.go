package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/redis/go-redis/v9"

	"github.com/meytardzhevtd/CodeTest/checker/internal/grpc/judgepb"
)

const testCacheKeyPrefix = "testcache:"

// cachedTestCase is the JSON-serializable mirror of judgepb.TestCase used for
// the Redis payload — kept separate from the generated proto type so cache
// (de)serialization doesn't depend on protobuf's internal struct layout.
type cachedTestCase struct {
	Number         int32  `json:"n"`
	Input          string `json:"i"`
	ExpectedOutput string `json:"o"`
}

// TestCache is a best-effort Redis-backed cache of a task's test cases,
// fronting the MinIO reads in loadTestCases. Redis is run with maxmemory +
// allkeys-lru (see docker-compose.yml), so eviction under capacity pressure
// is handled by Redis itself on every key access — this type only needs to
// read and write whole-task entries, with no TTL and no manual eviction
// bookkeeping.
//
// Every method treats a Redis error as a cache miss rather than a failure:
// judging must keep working off MinIO even if Redis is unreachable.
//
// There is no invalidation path: a task's tests are treated as immutable
// once uploaded (MVP scope). The upload endpoint (server/internal/tasks) can
// in fact replace a task's tests in MinIO, but that's a different process
// with no channel to the coordinator today — a re-upload after a task has
// been judged at least once will keep serving the stale cached copy until
// Redis evicts it under memory pressure. Wiring real invalidation (e.g. a
// Kafka event from the upload path) is deferred until this is a real need.
type TestCache struct {
	client *redis.Client
}

func NewTestCache(client *redis.Client) *TestCache {
	return &TestCache{client: client}
}

func (c *TestCache) Get(ctx context.Context, taskID string) ([]*judgepb.TestCase, bool) {
	raw, err := c.client.Get(ctx, testCacheKeyPrefix+taskID).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("[coordinator] redis get failed for task %s, falling back to MinIO: %v", taskID, err)
		}
		return nil, false
	}

	var cached []cachedTestCase
	if err := json.Unmarshal(raw, &cached); err != nil {
		log.Printf("[coordinator] corrupt redis cache entry for task %s, falling back to MinIO: %v", taskID, err)
		return nil, false
	}

	cases := make([]*judgepb.TestCase, len(cached))
	for i, tc := range cached {
		cases[i] = &judgepb.TestCase{Number: tc.Number, Input: tc.Input, ExpectedOutput: tc.ExpectedOutput}
	}
	return cases, true
}

func (c *TestCache) Set(ctx context.Context, taskID string, cases []*judgepb.TestCase) {
	cached := make([]cachedTestCase, len(cases))
	for i, tc := range cases {
		cached[i] = cachedTestCase{Number: tc.Number, Input: tc.Input, ExpectedOutput: tc.ExpectedOutput}
	}

	raw, err := json.Marshal(cached)
	if err != nil {
		log.Printf("[coordinator] failed to marshal test cases for task %s, skipping cache write: %v", taskID, err)
		return
	}

	if err := c.client.Set(ctx, testCacheKeyPrefix+taskID, raw, 0).Err(); err != nil {
		log.Printf("[coordinator] redis set failed for task %s: %v", taskID, err)
	}
}

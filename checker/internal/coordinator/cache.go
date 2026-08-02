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

type cachedTestCase struct {
	Number         int32  `json:"n"`
	Input          string `json:"i"`
	ExpectedOutput string `json:"o"`
}

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

package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type FeedCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewFeedCache(rdb *redis.Client) *FeedCache {
	return &FeedCache{rdb: rdb, ttl: 60 * time.Second}
}

func (c *FeedCache) InvalidateFeed(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = "feed:" + id
	}
	return c.rdb.Del(ctx, keys...).Err()
}

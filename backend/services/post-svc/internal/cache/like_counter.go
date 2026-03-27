package cache

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/usedcvnt/microtwitter/post-svc/internal/repository"
)

type LikeCounter struct {
	rdb      *redis.Client
	postRepo repository.PostRepository
	likeRepo repository.LikeRepository
	log      *slog.Logger
	dirty    map[string]int
	mu       sync.Mutex
}

func NewLikeCounter(rdb *redis.Client, postRepo repository.PostRepository, likeRepo repository.LikeRepository, log *slog.Logger) *LikeCounter {
	return &LikeCounter{
		rdb:      rdb,
		postRepo: postRepo,
		likeRepo: likeRepo,
		log:      log,
		dirty:    make(map[string]int),
	}
}

func (c *LikeCounter) Increment(ctx context.Context, postID string) (int64, error) {
	key := "post:likes:" + postID
	val, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	c.rdb.Expire(ctx, key, 300*time.Second)
	c.mu.Lock()
	c.dirty[postID] += 1
	c.mu.Unlock()
	return val, nil
}

func (c *LikeCounter) Decrement(ctx context.Context, postID string) (int64, error) {
	key := "post:likes:" + postID
	val, err := c.rdb.Decr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	c.rdb.Expire(ctx, key, 300*time.Second)
	c.mu.Lock()
	c.dirty[postID] -= 1
	c.mu.Unlock()
	return val, nil
}

func (c *LikeCounter) Get(ctx context.Context, postID string) (int64, error) {
	key := "post:likes:" + postID
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Count directly from likes table for accuracy
			count, err := c.likeRepo.CountByPost(ctx, postID)
			if err != nil {
				return 0, err
			}
			c.rdb.Set(ctx, key, count, 300*time.Second)
			return count, nil
		}
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (c *LikeCounter) StartSyncLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.flush(context.Background())
			return
		case <-ticker.C:
			c.flush(ctx)
		}
	}
}

func (c *LikeCounter) flush(ctx context.Context) {
	c.mu.Lock()
	if len(c.dirty) == 0 {
		c.mu.Unlock()
		return
	}
	snapshot := c.dirty
	c.dirty = make(map[string]int)
	c.mu.Unlock()

	for postID, delta := range snapshot {
		if delta == 0 {
			continue
		}
		if err := c.postRepo.IncrementLikes(ctx, postID, delta); err != nil {
			c.log.Error("failed to sync like counter", "post_id", postID, "delta", delta, "error", err)
			c.mu.Lock()
			c.dirty[postID] += delta
			c.mu.Unlock()
		}
	}
}

package cache

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/usedcvnt/microtwitter/post-svc/internal/repository"
)

type ViewCounter struct {
	rdb      *redis.Client
	postRepo repository.PostRepository
	log      *slog.Logger
	dirty    map[string]int
	mu       sync.Mutex
}

func NewViewCounter(rdb *redis.Client, postRepo repository.PostRepository, log *slog.Logger) *ViewCounter {
	return &ViewCounter{
		rdb:      rdb,
		postRepo: postRepo,
		log:      log,
		dirty:    make(map[string]int),
	}
}

// RecordView adds a viewer to the post's viewer set.
// Returns true if this is a new unique view.
func (c *ViewCounter) RecordView(ctx context.Context, postID, viewerID string) (bool, error) {
	key := "post:viewers:" + postID
	added, err := c.rdb.SAdd(ctx, key, viewerID).Result()
	if err != nil {
		return false, err
	}
	c.rdb.Expire(ctx, key, 7*24*time.Hour)

	if added > 0 {
		c.mu.Lock()
		c.dirty[postID]++
		c.mu.Unlock()
		return true, nil
	}
	return false, nil
}

// RecordViews records views for multiple posts from a single viewer.
func (c *ViewCounter) RecordViews(ctx context.Context, postIDs []string, viewerID string) {
	for _, postID := range postIDs {
		if _, err := c.RecordView(ctx, postID, viewerID); err != nil {
			c.log.Error("record view failed", "post_id", postID, "error", err)
		}
	}
}

// GetManyCounts returns view counts from Redis SETs for multiple posts.
func (c *ViewCounter) GetManyCounts(ctx context.Context, postIDs []string) map[string]int64 {
	if len(postIDs) == 0 {
		return nil
	}

	result := make(map[string]int64, len(postIDs))
	pipe := c.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, len(postIDs))
	for i, id := range postIDs {
		cmds[i] = pipe.SCard(ctx, "post:viewers:"+id)
	}
	_, _ = pipe.Exec(ctx)

	for i, id := range postIDs {
		if count, err := cmds[i].Result(); err == nil && count > 0 {
			result[id] = count
		}
	}
	return result
}

func (c *ViewCounter) StartSyncLoop(ctx context.Context, interval time.Duration) {
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

func (c *ViewCounter) flush(ctx context.Context) {
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
		if err := c.postRepo.IncrementViews(ctx, postID, delta); err != nil {
			c.log.Error("failed to sync view counter", "post_id", postID, "delta", delta, "error", err)
			c.mu.Lock()
			c.dirty[postID] += delta
			c.mu.Unlock()
		}
	}
}

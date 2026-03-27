package ws

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type Publisher struct {
	rdb *redis.Client
}

func NewPublisher(rdb *redis.Client) *Publisher {
	return &Publisher{rdb: rdb}
}

func (p *Publisher) PublishNotification(ctx context.Context, userID string, data any) {
	payload, err := json.Marshal(Event{Type: "notification", Data: mustMarshal(data)})
	if err != nil {
		return
	}
	p.rdb.Publish(ctx, "ws:channels:"+userID, payload)
}

func (p *Publisher) PublishNewPost(ctx context.Context, followerIDs []string, data any) {
	payload, err := json.Marshal(Event{Type: "new_post", Data: mustMarshal(data)})
	if err != nil {
		return
	}
	for _, id := range followerIDs {
		p.rdb.Publish(ctx, "ws:channels:"+id, payload)
	}
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Event struct {
	Type string `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client // userID -> client
	rdb     *redis.Client
	log     *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewHub(rdb *redis.Client, log *slog.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients: make(map[string]*Client),
		rdb:     rdb,
		log:     log,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	old, exists := h.clients[c.userID]
	h.clients[c.userID] = c
	h.mu.Unlock()

	if exists {
		old.Close()
	}

	go h.subscribe(c)
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if existing, ok := h.clients[c.userID]; ok && existing == c {
		delete(h.clients, c.userID)
	}
	h.mu.Unlock()
}

func (h *Hub) subscribe(c *Client) {
	channel := "ws:channels:" + c.userID
	sub := h.rdb.Subscribe(h.ctx, channel)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			c.Send([]byte(msg.Payload))
		case <-c.done:
			return
		case <-h.ctx.Done():
			return
		}
	}
}

func (h *Hub) Close() {
	h.cancel()
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		c.Close()
	}
}

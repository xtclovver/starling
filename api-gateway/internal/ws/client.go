package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 40 * time.Second
	pingPeriod = 30 * time.Second
)

type Client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	done   chan struct{}
	once   sync.Once
}

func NewClient(userID string, conn *websocket.Conn) *Client {
	return &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 64),
		done:   make(chan struct{}),
	}
}

func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}

func (c *Client) Close() {
	c.once.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister(c)
		c.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

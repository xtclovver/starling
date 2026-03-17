package ws

import (
	"log/slog"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	hub    *Hub
	secret []byte
	log    *slog.Logger
}

func NewHandler(hub *Hub, jwtSecret string, log *slog.Logger) *Handler {
	return &Handler{hub: hub, secret: []byte(jwtSecret), log: log}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userID, err := h.validateToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("websocket upgrade failed", "error", err)
		return
	}

	client := NewClient(userID, conn)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump(h.hub)
}

func (h *Handler) validateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return h.secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	return sub, nil
}

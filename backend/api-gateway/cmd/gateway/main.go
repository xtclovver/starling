package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/usedcvnt/microtwitter/api-gateway/internal/config"
	grpcclient "github.com/usedcvnt/microtwitter/api-gateway/internal/grpc_client"
	"github.com/usedcvnt/microtwitter/api-gateway/internal/handler"
	"github.com/usedcvnt/microtwitter/api-gateway/internal/middleware"
	"github.com/usedcvnt/microtwitter/api-gateway/internal/ws"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Redis
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Error("failed to parse redis url", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("failed to ping redis", "error", err)
		os.Exit(1)
	}

	// gRPC clients
	clients, err := grpcclient.New(cfg.UserSvcAddr, cfg.PostSvcAddr, cfg.CommentSvcAddr, cfg.MediaSvcAddr)
	if err != nil {
		log.Error("failed to create grpc clients", "error", err)
		os.Exit(1)
	}
	defer clients.Close()

	// Handlers
	authH := handler.NewAuthHandler(clients.User)
	userH := handler.NewUserHandler(clients.User, clients.Post)
	postH := handler.NewPostHandler(clients.Post, clients.User)
	commentH := handler.NewCommentHandler(clients.Comment)
	mediaH := handler.NewMediaHandler(clients.Media)

	// WebSocket
	hub := ws.NewHub(rdb, log)
	defer hub.Close()
	wsHandler := ws.NewHandler(hub, cfg.JWTSecret, log)

	// Middleware
	auth := middleware.NewAuth(cfg.JWTSecret)
	rl := middleware.NewRateLimiter(rdb)

	// Router (stdlib)
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /api/auth/register", authH.Register)
	mux.HandleFunc("POST /api/auth/login", authH.Login)
	mux.HandleFunc("POST /api/auth/refresh", authH.Refresh)

	// User routes - public
	mux.HandleFunc("GET /api/users/search", userH.SearchUsers)
	mux.HandleFunc("GET /api/users/{id}", userH.GetUser)
	mux.HandleFunc("GET /api/users/{id}/posts", userH.GetUserPosts)
	mux.HandleFunc("GET /api/users/{id}/followers", userH.GetFollowers)
	mux.HandleFunc("GET /api/users/{id}/following", userH.GetFollowing)

	// User routes - auth required
	mux.Handle("PUT /api/users/{id}", auth.Required(http.HandlerFunc(userH.UpdateUser)))
	mux.Handle("DELETE /api/users/{id}", auth.Required(http.HandlerFunc(userH.DeleteUser)))
	mux.Handle("POST /api/users/{id}/follow", auth.Required(http.HandlerFunc(userH.Follow)))
	mux.Handle("DELETE /api/users/{id}/follow", auth.Required(http.HandlerFunc(userH.Unfollow)))

	// Post routes - public
	mux.HandleFunc("GET /api/posts/{id}", postH.GetPost)

	// Post routes - auth required
	mux.Handle("POST /api/posts", auth.Required(http.HandlerFunc(postH.CreatePost)))
	mux.Handle("DELETE /api/posts/{id}", auth.Required(http.HandlerFunc(postH.DeletePost)))
	mux.Handle("GET /api/feed", auth.Required(http.HandlerFunc(postH.GetFeed)))
	mux.Handle("POST /api/posts/{id}/like", auth.Required(http.HandlerFunc(postH.LikePost)))
	mux.Handle("DELETE /api/posts/{id}/like", auth.Required(http.HandlerFunc(postH.UnlikePost)))

	// Comment routes
	mux.HandleFunc("GET /api/posts/{id}/comments", commentH.GetCommentTree)
	mux.Handle("POST /api/posts/{id}/comments", auth.Required(http.HandlerFunc(commentH.CreateComment)))
	mux.Handle("DELETE /api/comments/{id}", auth.Required(http.HandlerFunc(commentH.DeleteComment)))
	mux.Handle("POST /api/comments/{id}/like", auth.Required(http.HandlerFunc(commentH.LikeComment)))
	mux.Handle("DELETE /api/comments/{id}/like", auth.Required(http.HandlerFunc(commentH.UnlikeComment)))

	// Media
	mux.Handle("POST /api/upload", auth.Required(http.HandlerFunc(mediaH.Upload)))

	// WebSocket
	mux.Handle("GET /api/ws", wsHandler)

	// Middleware chain
	var h http.Handler = mux
	h = middleware.Logger(log)(h)
	h = middleware.SecurityHeaders(h)
	h = middleware.CORS(cfg.CORSOrigin)(h)
	h = rl.Guest()(h) // default rate limit
	h = middleware.BodyLimit(1 << 20)(h) // 1MB default
	h = middleware.RequestID(h)
	h = middleware.Recovery(log)(h)

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("starting api-gateway", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down api-gateway")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}

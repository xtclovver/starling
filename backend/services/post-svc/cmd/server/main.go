package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	"github.com/usedcvnt/microtwitter/post-svc/internal/cache"
	"github.com/usedcvnt/microtwitter/post-svc/internal/config"
	grpcserver "github.com/usedcvnt/microtwitter/post-svc/internal/grpc"
	"github.com/usedcvnt/microtwitter/post-svc/internal/repository"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DBUrl)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

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

	postRepo := repository.NewPostRepository(pool)
	likeRepo := repository.NewLikeRepository(pool)
	bookmarkRepo := repository.NewBookmarkRepository(pool)
	hashtagRepo := repository.NewHashtagRepository(pool)
	repostRepo := repository.NewRepostRepository(pool)
	likeCounter := cache.NewLikeCounter(rdb, postRepo, likeRepo, log)

	go likeCounter.StartSyncLoop(ctx, cfg.LikeSyncInterval)

	srv := grpc.NewServer()
	postServer := grpcserver.NewServer(postRepo, likeRepo, bookmarkRepo, hashtagRepo, repostRepo, likeCounter, log)
	pb.RegisterPostServiceServer(srv, postServer)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("post.v1.PostService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(srv)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info("starting post-svc", "port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			log.Error("grpc server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down post-svc")
	cancel()
	srv.GracefulStop()
}

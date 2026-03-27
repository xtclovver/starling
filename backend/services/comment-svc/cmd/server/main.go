package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	"github.com/usedcvnt/microtwitter/comment-svc/internal/config"
	grpcserver "github.com/usedcvnt/microtwitter/comment-svc/internal/grpc"
	"github.com/usedcvnt/microtwitter/comment-svc/internal/repository"
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

	commentRepo := repository.NewCommentRepository(pool)
	commentLikeRepo := repository.NewCommentLikeRepository(pool)

	srv := grpc.NewServer()
	commentServer := grpcserver.NewServer(commentRepo, commentLikeRepo, log)
	pb.RegisterCommentServiceServer(srv, commentServer)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("comment.v1.CommentService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(srv)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info("starting comment-svc", "port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			log.Error("grpc server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down comment-svc")
	cancel()
	srv.GracefulStop()
}

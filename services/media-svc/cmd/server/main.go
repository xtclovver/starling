package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "github.com/usedcvnt/microtwitter/gen/go/media/v1"
	"github.com/usedcvnt/microtwitter/media-svc/internal/config"
	grpcserver "github.com/usedcvnt/microtwitter/media-svc/internal/grpc"
	"github.com/usedcvnt/microtwitter/media-svc/internal/repository"
	"github.com/usedcvnt/microtwitter/media-svc/internal/storage"
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

	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Error("failed to create minio client", "error", err)
		os.Exit(1)
	}

	store := storage.NewMinIOClient(minioClient, cfg.MinIOBucket)
	if err := store.EnsureBucket(ctx); err != nil {
		log.Error("failed to ensure bucket", "error", err)
		os.Exit(1)
	}

	mediaRepo := repository.NewMediaRepository(pool)

	srv := grpc.NewServer()
	mediaServer := grpcserver.NewServer(mediaRepo, store, cfg.MinIOBucket, log)
	pb.RegisterMediaServiceServer(srv, mediaServer)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("media.v1.MediaService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(srv)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info("starting media-svc", "port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			log.Error("grpc server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down media-svc")
	srv.GracefulStop()
}

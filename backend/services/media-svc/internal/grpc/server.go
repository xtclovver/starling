package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/usedcvnt/microtwitter/gen/go/media/v1"
	"github.com/usedcvnt/microtwitter/media-svc/internal/model"
	"github.com/usedcvnt/microtwitter/media-svc/internal/repository"
	"github.com/usedcvnt/microtwitter/media-svc/internal/storage"
	"github.com/usedcvnt/microtwitter/media-svc/internal/validation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedMediaServiceServer
	mediaRepo      repository.MediaRepository
	storage        *storage.MinIOClient
	bucket         string
	publicEndpoint string
	log            *slog.Logger
}

func NewServer(
	mediaRepo repository.MediaRepository,
	storage *storage.MinIOClient,
	bucket string,
	publicEndpoint string,
	log *slog.Logger,
) *Server {
	return &Server{
		mediaRepo:      mediaRepo,
		storage:        storage,
		bucket:         bucket,
		publicEndpoint: publicEndpoint,
		log:            log,
	}
}

func (s *Server) UploadMedia(ctx context.Context, req *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("UploadMedia", "duration", time.Since(start)) }()

	if err := validation.ValidateFile(req.GetContentType(), req.GetData()); err != nil {
		if errors.Is(err, validation.ErrUnsupportedType) {
			return nil, status.Error(codes.InvalidArgument, "unsupported content type")
		}
		if errors.Is(err, validation.ErrFileTooLarge) {
			return nil, status.Error(codes.InvalidArgument, "file too large (max 10MB)")
		}
		if errors.Is(err, validation.ErrInvalidMagic) {
			return nil, status.Error(codes.InvalidArgument, "file content does not match declared type")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	objectKey := validation.GenerateObjectKey(req.GetUserId(), req.GetContentType())

	if err := s.storage.Upload(ctx, objectKey, req.GetData(), req.GetContentType()); err != nil {
		s.log.Error("upload to minio failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	var postID *string
	if req.GetPostId() != "" {
		pid := req.GetPostId()
		postID = &pid
	}

	media, err := s.mediaRepo.Create(ctx, req.GetUserId(), postID, s.bucket, objectKey, req.GetContentType())
	if err != nil {
		s.log.Error("create media record failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	url := fmt.Sprintf("%s/%s/%s", s.publicEndpoint, s.bucket, objectKey)

	return &pb.UploadMediaResponse{
		Media: toProtoMedia(media, url),
	}, nil
}

func (s *Server) GetPresignedUploadURL(ctx context.Context, req *pb.GetPresignedUploadURLRequest) (*pb.GetPresignedUploadURLResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetPresignedUploadURL", "duration", time.Since(start)) }()

	objectKey := validation.GenerateObjectKey(req.GetUserId(), req.GetContentType())

	uploadURL, err := s.storage.GetPresignedUploadURL(ctx, objectKey, 15*time.Minute)
	if err != nil {
		s.log.Error("get presigned upload url failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.GetPresignedUploadURLResponse{
		UploadUrl: uploadURL,
		ObjectKey: objectKey,
	}, nil
}

func (s *Server) GetMediaURL(ctx context.Context, req *pb.GetMediaURLRequest) (*pb.GetMediaURLResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetMediaURL", "duration", time.Since(start)) }()

	url := fmt.Sprintf("%s/%s/%s", s.publicEndpoint, s.bucket, req.GetObjectKey())
	return &pb.GetMediaURLResponse{Url: url}, nil
}

func (s *Server) DeleteMedia(ctx context.Context, req *pb.DeleteMediaRequest) (*pb.DeleteMediaResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("DeleteMedia", "duration", time.Since(start)) }()

	media, err := s.mediaRepo.Delete(ctx, req.GetId(), req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		if errors.Is(err, repository.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, "not media owner")
		}
		s.log.Error("delete media failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := s.storage.Delete(ctx, media.ObjectKey); err != nil {
		s.log.Error("delete from minio failed", "error", err)
	}

	return &pb.DeleteMediaResponse{}, nil
}

func toProtoMedia(m *model.Media, url string) *pb.Media {
	pm := &pb.Media{
		Id:          m.ID,
		UserId:      m.UserID,
		Bucket:      m.Bucket,
		ObjectKey:   m.ObjectKey,
		ContentType: m.ContentType,
		Url:         url,
		CreatedAt:   timestamppb.New(m.CreatedAt),
	}
	if m.PostID != nil {
		pm.PostId = *m.PostID
	}
	return pm
}

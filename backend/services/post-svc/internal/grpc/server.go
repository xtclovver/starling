package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	pb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	"github.com/usedcvnt/microtwitter/post-svc/internal/cache"
	"github.com/usedcvnt/microtwitter/post-svc/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedPostServiceServer
	postRepo    repository.PostRepository
	likeRepo    repository.LikeRepository
	likeCounter *cache.LikeCounter
	log         *slog.Logger
}

func NewServer(
	postRepo repository.PostRepository,
	likeRepo repository.LikeRepository,
	likeCounter *cache.LikeCounter,
	log *slog.Logger,
) *Server {
	return &Server{
		postRepo:    postRepo,
		likeRepo:    likeRepo,
		likeCounter: likeCounter,
		log:         log,
	}
}

func (s *Server) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("CreatePost", "duration", time.Since(start)) }()

	content := req.GetContent()
	if content == "" || len(content) > 280 {
		return nil, status.Error(codes.InvalidArgument, "content must be 1-280 characters")
	}

	post, err := s.postRepo.Create(ctx, req.GetUserId(), content, req.GetMediaUrl())
	if err != nil {
		s.log.Error("create post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.CreatePostResponse{Post: toProtoPost(post)}, nil
}

func (s *Server) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetPost", "duration", time.Since(start)) }()

	post, err := s.postRepo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		s.log.Error("get post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Try to get fresh like count from Redis
	if count, err := s.likeCounter.Get(ctx, post.ID); err == nil {
		post.LikesCount = count
	}

	return &pb.GetPostResponse{Post: toProtoPost(post)}, nil
}

func (s *Server) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("DeletePost", "duration", time.Since(start)) }()

	if err := s.postRepo.SoftDelete(ctx, req.GetId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		if errors.Is(err, repository.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, "not post owner")
		}
		s.log.Error("delete post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.DeletePostResponse{}, nil
}

func (s *Server) GetFeed(ctx context.Context, req *pb.GetFeedRequest) (*pb.GetFeedResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetFeed", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	posts, nextCursor, hasMore, err := s.postRepo.GetFeed(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get feed failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, p := range posts {
		p := p
		pbPosts[i] = toProtoPost(&p)
	}

	return &pb.GetFeedResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) LikePost(ctx context.Context, req *pb.LikePostRequest) (*pb.LikePostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("LikePost", "duration", time.Since(start)) }()

	if err := s.likeRepo.LikePost(ctx, req.GetPostId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrAlreadyLiked) {
			return &pb.LikePostResponse{}, nil
		}
		s.log.Error("like post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if _, err := s.likeCounter.Increment(ctx, req.GetPostId()); err != nil {
		s.log.Error("increment like counter failed", "error", err)
	}

	return &pb.LikePostResponse{}, nil
}

func (s *Server) UnlikePost(ctx context.Context, req *pb.UnlikePostRequest) (*pb.UnlikePostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("UnlikePost", "duration", time.Since(start)) }()

	if err := s.likeRepo.UnlikePost(ctx, req.GetPostId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &pb.UnlikePostResponse{}, nil
		}
		s.log.Error("unlike post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if _, err := s.likeCounter.Decrement(ctx, req.GetPostId()); err != nil {
		s.log.Error("decrement like counter failed", "error", err)
	}

	return &pb.UnlikePostResponse{}, nil
}

func (s *Server) GetPostsByUser(ctx context.Context, req *pb.GetPostsByUserRequest) (*pb.GetPostsByUserResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetPostsByUser", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	posts, nextCursor, hasMore, err := s.postRepo.GetByUser(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get posts by user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, p := range posts {
		p := p
		pbPosts[i] = toProtoPost(&p)
	}

	return &pb.GetPostsByUserResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

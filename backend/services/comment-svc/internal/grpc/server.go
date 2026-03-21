package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	pb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	"github.com/usedcvnt/microtwitter/comment-svc/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedCommentServiceServer
	commentRepo     repository.CommentRepository
	commentLikeRepo repository.CommentLikeRepository
	log             *slog.Logger
}

func NewServer(
	commentRepo repository.CommentRepository,
	commentLikeRepo repository.CommentLikeRepository,
	log *slog.Logger,
) *Server {
	return &Server{
		commentRepo:     commentRepo,
		commentLikeRepo: commentLikeRepo,
		log:             log,
	}
}

func (s *Server) CreateComment(ctx context.Context, req *pb.CreateCommentRequest) (*pb.CreateCommentResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("CreateComment", "duration", time.Since(start)) }()

	content := req.GetContent()
	if content == "" || len(content) > 500 {
		return nil, status.Error(codes.InvalidArgument, "content must be 1-500 characters")
	}

	var parentID *string
	if req.GetParentId() != "" {
		pid := req.GetParentId()
		parentID = &pid
	}

	comment, err := s.commentRepo.Create(ctx, req.GetPostId(), req.GetUserId(), parentID, content)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "parent comment not found")
		}
		if errors.Is(err, repository.ErrMaxDepth) {
			return nil, status.Error(codes.InvalidArgument, "maximum nesting depth exceeded")
		}
		s.log.Error("create comment failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := s.commentRepo.IncrementPostComments(ctx, req.GetPostId()); err != nil {
		s.log.Error("increment post comments failed", "error", err)
	}

	return &pb.CreateCommentResponse{Comment: toProtoComment(comment)}, nil
}

func (s *Server) GetCommentTree(ctx context.Context, req *pb.GetCommentTreeRequest) (*pb.GetCommentTreeResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetCommentTree", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	comments, nextCursor, err := s.commentRepo.GetTree(ctx, req.GetPostId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get comment tree failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbComments := make([]*pb.Comment, len(comments))
	for i, c := range comments {
		c := c
		pbComments[i] = toProtoComment(&c)
	}

	if uid := req.GetUserId(); uid != "" {
		ids := collectCommentIDs(pbComments)
		if len(ids) > 0 {
			likedMap, err := s.commentLikeRepo.AreLikedComments(ctx, ids, uid)
			if err != nil {
				s.log.Error("check liked comments failed", "error", err)
			} else {
				setLikedFlags(pbComments, likedMap)
			}
		}
	}

	return &pb.GetCommentTreeResponse{
		Comments: pbComments,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
		},
	}, nil
}

func (s *Server) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("DeleteComment", "duration", time.Since(start)) }()

	// Get comment to know post_id for counter update
	if err := s.commentRepo.SoftDelete(ctx, req.GetId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "comment not found")
		}
		if errors.Is(err, repository.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, "not comment owner")
		}
		s.log.Error("delete comment failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.DeleteCommentResponse{}, nil
}

func (s *Server) LikeComment(ctx context.Context, req *pb.LikeCommentRequest) (*pb.LikeCommentResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("LikeComment", "duration", time.Since(start)) }()

	if err := s.commentLikeRepo.Like(ctx, req.GetCommentId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrAlreadyLiked) {
			return &pb.LikeCommentResponse{}, nil
		}
		s.log.Error("like comment failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := s.commentLikeRepo.IncrementLikes(ctx, req.GetCommentId(), 1); err != nil {
		s.log.Error("increment comment likes failed", "error", err)
	}

	return &pb.LikeCommentResponse{}, nil
}

func (s *Server) UnlikeComment(ctx context.Context, req *pb.UnlikeCommentRequest) (*pb.UnlikeCommentResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("UnlikeComment", "duration", time.Since(start)) }()

	if err := s.commentLikeRepo.Unlike(ctx, req.GetCommentId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &pb.UnlikeCommentResponse{}, nil
		}
		s.log.Error("unlike comment failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := s.commentLikeRepo.IncrementLikes(ctx, req.GetCommentId(), -1); err != nil {
		s.log.Error("decrement comment likes failed", "error", err)
	}

	return &pb.UnlikeCommentResponse{}, nil
}

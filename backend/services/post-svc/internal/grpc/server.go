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
	postRepo     repository.PostRepository
	likeRepo     repository.LikeRepository
	bookmarkRepo repository.BookmarkRepository
	hashtagRepo  repository.HashtagRepository
	repostRepo   repository.RepostRepository
	likeCounter  *cache.LikeCounter
	viewCounter  *cache.ViewCounter
	log          *slog.Logger
}

func NewServer(
	postRepo repository.PostRepository,
	likeRepo repository.LikeRepository,
	bookmarkRepo repository.BookmarkRepository,
	hashtagRepo repository.HashtagRepository,
	repostRepo repository.RepostRepository,
	likeCounter *cache.LikeCounter,
	viewCounter *cache.ViewCounter,
	log *slog.Logger,
) *Server {
	return &Server{
		postRepo:     postRepo,
		likeRepo:     likeRepo,
		bookmarkRepo: bookmarkRepo,
		hashtagRepo:  hashtagRepo,
		repostRepo:   repostRepo,
		likeCounter:  likeCounter,
		viewCounter:  viewCounter,
		log:          log,
	}
}

func (s *Server) enrichPosts(ctx context.Context, posts []*pb.Post, viewerID string) {
	if len(posts) == 0 {
		return
	}

	postIDs := make([]string, len(posts))
	for i, p := range posts {
		postIDs[i] = p.GetId()
	}

	// Load hashtags for all posts
	tagsMap, err := s.hashtagRepo.GetTagsByPostIDs(ctx, postIDs)
	if err == nil {
		for _, p := range posts {
			if tags, ok := tagsMap[p.GetId()]; ok {
				p.Hashtags = tags
			}
		}
	}

	// Refresh like counts from Redis (source of truth)
	likeCounts := s.likeCounter.GetMany(ctx, postIDs)
	if likeCounts != nil {
		for _, p := range posts {
			if count, ok := likeCounts[p.GetId()]; ok {
				p.LikesCount = safeInt32(count)
			}
		}
	}

	// Refresh view counts from Redis
	viewCounts := s.viewCounter.GetManyCounts(ctx, postIDs)
	if viewCounts != nil {
		for _, p := range posts {
			if count, ok := viewCounts[p.GetId()]; ok && count > int64(p.GetViewsCount()) {
				p.ViewsCount = safeInt32(count)
			}
		}
	}

	if viewerID == "" {
		return
	}

	// Load user-specific flags
	likedMap, _ := s.likeRepo.AreLiked(ctx, postIDs, viewerID)
	bookmarkedMap, _ := s.bookmarkRepo.AreBookmarked(ctx, postIDs, viewerID)
	repostedMap, _ := s.repostRepo.AreReposted(ctx, postIDs, viewerID)

	for _, p := range posts {
		id := p.GetId()
		if likedMap != nil {
			p.Liked = likedMap[id]
		}
		if bookmarkedMap != nil {
			p.Bookmarked = bookmarkedMap[id]
		}
		if repostedMap != nil {
			p.Reposted = repostedMap[id]
		}
	}
}

func (s *Server) loadHashtags(ctx context.Context, post *pb.Post) {
	tags, err := s.hashtagRepo.GetTagsByPostID(ctx, post.GetId())
	if err == nil && len(tags) > 0 {
		post.Hashtags = tags
	}
}

func (s *Server) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("CreatePost", "duration", time.Since(start)) }()

	content := req.GetContent()
	if content == "" || len(content) > 280 {
		return nil, status.Error(codes.InvalidArgument, "content must be 1-280 characters")
	}

	if len(req.GetMediaUrls()) > 10 {
		return nil, status.Error(codes.InvalidArgument, "max 10 media per post")
	}

	post, err := s.postRepo.Create(ctx, req.GetUserId(), content)
	if err != nil {
		s.log.Error("create post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if tags := repository.ExtractHashtags(content); len(tags) > 0 {
		if err := s.hashtagRepo.UpsertAndLink(ctx, post.ID, tags); err != nil {
			s.log.Error("link hashtags failed", "error", err)
		}
	}

	protoPost := toProtoPost(post)
	s.loadHashtags(ctx, protoPost)
	return &pb.CreatePostResponse{Post: protoPost}, nil
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

	protoPost := toProtoPost(post)
	s.enrichPosts(ctx, []*pb.Post{protoPost}, req.GetUserId())
	return &pb.GetPostResponse{Post: protoPost}, nil
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

	s.enrichPosts(ctx, pbPosts, req.GetUserId())

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

	s.enrichPosts(ctx, pbPosts, req.GetViewerId())

	return &pb.GetPostsByUserResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) GetGlobalFeed(ctx context.Context, req *pb.GetGlobalFeedRequest) (*pb.GetGlobalFeedResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetGlobalFeed", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	posts, nextCursor, hasMore, err := s.postRepo.GetGlobalFeed(ctx, cursor, int(limit))
	if err != nil {
		s.log.Error("get global feed failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, p := range posts {
		pbPosts[i] = toProtoPost(&p)
	}

	s.enrichPosts(ctx, pbPosts, req.GetUserId())

	return &pb.GetGlobalFeedResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) BookmarkPost(ctx context.Context, req *pb.BookmarkPostRequest) (*pb.BookmarkPostResponse, error) {
	if err := s.bookmarkRepo.Bookmark(ctx, req.GetPostId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrAlreadyBookmarked) {
			return &pb.BookmarkPostResponse{}, nil
		}
		s.log.Error("bookmark post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.BookmarkPostResponse{}, nil
}

func (s *Server) UnbookmarkPost(ctx context.Context, req *pb.UnbookmarkPostRequest) (*pb.UnbookmarkPostResponse, error) {
	if err := s.bookmarkRepo.Unbookmark(ctx, req.GetPostId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &pb.UnbookmarkPostResponse{}, nil
		}
		s.log.Error("unbookmark post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.UnbookmarkPostResponse{}, nil
}

func (s *Server) GetBookmarks(ctx context.Context, req *pb.GetBookmarksRequest) (*pb.GetBookmarksResponse, error) {
	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	posts, nextCursor, hasMore, err := s.bookmarkRepo.GetByUser(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get bookmarks failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, p := range posts {
		pbPosts[i] = toProtoPost(&p)
	}

	s.enrichPosts(ctx, pbPosts, req.GetUserId())

	return &pb.GetBookmarksResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("UpdatePost", "duration", time.Since(start)) }()

	content := req.GetContent()
	if content == "" || len(content) > 280 {
		return nil, status.Error(codes.InvalidArgument, "content must be 1-280 characters")
	}

	post, err := s.postRepo.Update(ctx, req.GetId(), req.GetUserId(), content)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "post not found")
		}
		if errors.Is(err, repository.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, "not post owner")
		}
		s.log.Error("update post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Re-link hashtags
	_ = s.hashtagRepo.UnlinkAll(ctx, post.ID)
	if tags := repository.ExtractHashtags(content); len(tags) > 0 {
		if err := s.hashtagRepo.UpsertAndLink(ctx, post.ID, tags); err != nil {
			s.log.Error("re-link hashtags failed", "error", err)
		}
	}

	protoPost := toProtoPost(post)
	s.enrichPosts(ctx, []*pb.Post{protoPost}, req.GetUserId())
	return &pb.UpdatePostResponse{Post: protoPost}, nil
}

func (s *Server) GetPostsByHashtag(ctx context.Context, req *pb.GetPostsByHashtagRequest) (*pb.GetPostsByHashtagResponse, error) {
	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	posts, nextCursor, hasMore, err := s.hashtagRepo.GetPostsByHashtag(ctx, req.GetTag(), cursor, int(limit))
	if err != nil {
		s.log.Error("get posts by hashtag failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, p := range posts {
		pbPosts[i] = toProtoPost(&p)
	}

	s.enrichPosts(ctx, pbPosts, req.GetUserId())

	return &pb.GetPostsByHashtagResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) GetTrendingHashtags(ctx context.Context, req *pb.GetTrendingHashtagsRequest) (*pb.GetTrendingHashtagsResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}

	trending, err := s.hashtagRepo.GetTrending(ctx, limit)
	if err != nil {
		s.log.Error("get trending hashtags failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbTrending := make([]*pb.TrendingHashtag, len(trending))
	for i, t := range trending {
		pbTrending[i] = &pb.TrendingHashtag{Tag: t.Tag, PostCount: t.PostCount}
	}

	return &pb.GetTrendingHashtagsResponse{Hashtags: pbTrending}, nil
}

func (s *Server) RepostPost(ctx context.Context, req *pb.RepostPostRequest) (*pb.RepostPostResponse, error) {
	if err := s.repostRepo.Repost(ctx, req.GetPostId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrAlreadyReposted) {
			return &pb.RepostPostResponse{}, nil
		}
		s.log.Error("repost failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	_ = s.postRepo.IncrementReposts(ctx, req.GetPostId(), 1)
	return &pb.RepostPostResponse{}, nil
}

func (s *Server) UnrepostPost(ctx context.Context, req *pb.UnrepostPostRequest) (*pb.UnrepostPostResponse, error) {
	if err := s.repostRepo.Unrepost(ctx, req.GetPostId(), req.GetUserId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &pb.UnrepostPostResponse{}, nil
		}
		s.log.Error("unrepost failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	_ = s.postRepo.IncrementReposts(ctx, req.GetPostId(), -1)
	return &pb.UnrepostPostResponse{}, nil
}

func (s *Server) QuotePost(ctx context.Context, req *pb.QuotePostRequest) (*pb.QuotePostResponse, error) {
	content := req.GetContent()
	if content == "" || len(content) > 280 {
		return nil, status.Error(codes.InvalidArgument, "content must be 1-280 characters")
	}

	_, err := s.repostRepo.QuotePost(ctx, req.GetPostId(), req.GetUserId(), content)
	if err != nil {
		s.log.Error("quote post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Create a new post for the quote
	post, err := s.postRepo.Create(ctx, req.GetUserId(), content)
	if err != nil {
		s.log.Error("create quote post failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	_ = s.postRepo.IncrementReposts(ctx, req.GetPostId(), 1)

	if tags := repository.ExtractHashtags(content); len(tags) > 0 {
		_ = s.hashtagRepo.UpsertAndLink(ctx, post.ID, tags)
	}

	protoPost := toProtoPost(post)
	s.enrichPosts(ctx, []*pb.Post{protoPost}, req.GetUserId())
	return &pb.QuotePostResponse{Post: protoPost}, nil
}

func (s *Server) GetRepostsByUser(ctx context.Context, req *pb.GetRepostsByUserRequest) (*pb.GetRepostsByUserResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetRepostsByUser", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	posts, nextCursor, hasMore, err := s.repostRepo.GetRepostedPostsByUser(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get reposts by user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbPosts := make([]*pb.Post, len(posts))
	for i, p := range posts {
		p := p
		pbPosts[i] = toProtoPost(&p)
	}

	s.enrichPosts(ctx, pbPosts, req.GetViewerId())

	return &pb.GetRepostsByUserResponse{
		Posts: pbPosts,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) RecordViews(ctx context.Context, req *pb.RecordViewsRequest) (*pb.RecordViewsResponse, error) {
	if len(req.GetPostIds()) == 0 {
		return &pb.RecordViewsResponse{}, nil
	}
	if len(req.GetPostIds()) > 50 {
		return nil, status.Error(codes.InvalidArgument, "max 50 posts per request")
	}

	viewerID := req.GetViewerId()
	if viewerID == "" {
		return &pb.RecordViewsResponse{}, nil
	}

	s.viewCounter.RecordViews(ctx, req.GetPostIds(), viewerID)
	return &pb.RecordViewsResponse{}, nil
}

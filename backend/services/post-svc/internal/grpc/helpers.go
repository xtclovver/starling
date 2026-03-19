package grpc

import (
	"math"

	pb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	"github.com/usedcvnt/microtwitter/post-svc/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func safeInt32(v int64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

func toProtoPost(p *model.Post) *pb.Post {
	post := &pb.Post{
		Id:            p.ID,
		UserId:        p.UserID,
		Content:       p.Content,
		MediaUrl:      p.MediaURL,
		LikesCount:    safeInt32(p.LikesCount),
		CommentsCount: safeInt32(p.CommentsCount),
		RepostsCount:  safeInt32(p.RepostsCount),
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
	if p.EditedAt != nil {
		post.EditedAt = timestamppb.New(*p.EditedAt)
	}
	return post
}

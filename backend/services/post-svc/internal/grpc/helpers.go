package grpc

import (
	pb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	"github.com/usedcvnt/microtwitter/post-svc/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoPost(p *model.Post) *pb.Post {
	return &pb.Post{
		Id:            p.ID,
		UserId:        p.UserID,
		Content:       p.Content,
		MediaUrl:      p.MediaURL,
		LikesCount:    int32(p.LikesCount),
		CommentsCount: int32(p.CommentsCount),
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
}

package grpc

import (
	pb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	"github.com/usedcvnt/microtwitter/comment-svc/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoComment(c *model.Comment) *pb.Comment {
	pc := &pb.Comment{
		Id:         c.ID,
		PostId:     c.PostID,
		UserId:     c.UserID,
		Content:    c.Content,
		LikesCount: c.LikesCount,
		Depth:      c.Depth,
		CreatedAt:  timestamppb.New(c.CreatedAt),
		UpdatedAt:  timestamppb.New(c.UpdatedAt),
	}
	if c.ParentID != nil {
		pc.ParentId = *c.ParentID
	}
	for _, child := range c.Children {
		pc.Children = append(pc.Children, toProtoComment(child))
	}
	return pc
}

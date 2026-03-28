package grpc

import (
	pb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	"github.com/usedcvnt/microtwitter/comment-svc/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func collectCommentIDs(comments []*pb.Comment) []string {
	var ids []string
	for _, c := range comments {
		ids = append(ids, c.GetId())
		ids = append(ids, collectCommentIDs(c.GetChildren())...)
	}
	return ids
}

func setLikedFlags(comments []*pb.Comment, likedMap map[string]bool) {
	for _, c := range comments {
		c.Liked = likedMap[c.GetId()]
		setLikedFlags(c.GetChildren(), likedMap)
	}
}

func toProtoComment(c *model.Comment) *pb.Comment {
	pc := &pb.Comment{
		Id:         c.ID,
		PostId:     c.PostID,
		UserId:     c.UserID,
		Content:    c.Content,
		MediaUrl:   c.MediaURL,
		LikesCount: c.LikesCount,
		Depth:      c.Depth,
		CreatedAt:  timestamppb.New(c.CreatedAt),
		UpdatedAt:  timestamppb.New(c.UpdatedAt),
	}
	if c.EditedAt != nil {
		pc.EditedAt = timestamppb.New(*c.EditedAt)
	}
	if c.ParentID != nil {
		pc.ParentId = *c.ParentID
	}
	for _, child := range c.Children {
		pc.Children = append(pc.Children, toProtoComment(child))
	}
	return pc
}

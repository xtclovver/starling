package grpc

import (
	"strings"

	pb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
	"github.com/usedcvnt/microtwitter/user-svc/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoUser(u *model.User) *pb.User {
	return &pb.User{
		Id:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		AvatarUrl:   u.AvatarURL,
		BannerUrl:   u.BannerURL,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

func toProtoNotification(n *model.Notification) *pb.Notification {
	pn := &pb.Notification{
		Id:        n.ID,
		UserId:    n.UserID,
		ActorId:   n.ActorID,
		Type:      n.Type,
		Read:      n.Read,
		CreatedAt: timestamppb.New(n.CreatedAt),
	}
	if n.PostID != nil {
		pn.PostId = *n.PostID
	}
	if n.CommentID != nil {
		pn.CommentId = *n.CommentID
	}
	return pn
}

func isValidEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at < 1 {
		return false
	}
	dot := strings.LastIndexByte(email[at:], '.')
	return dot > 1 && dot < len(email[at:])-1
}

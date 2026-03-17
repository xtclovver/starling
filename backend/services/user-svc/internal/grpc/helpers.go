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
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

func isValidEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at < 1 {
		return false
	}
	dot := strings.LastIndexByte(email[at:], '.')
	return dot > 1 && dot < len(email[at:])-1
}

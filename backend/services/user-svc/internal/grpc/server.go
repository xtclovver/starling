package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	pb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
	"github.com/usedcvnt/microtwitter/user-svc/internal/auth"
	"github.com/usedcvnt/microtwitter/user-svc/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedUserServiceServer
	userRepo   repository.UserRepository
	followRepo repository.FollowRepository
	notifRepo  repository.NotificationRepository
	jwt        *auth.JWTManager
	log        *slog.Logger
}

func NewServer(
	userRepo repository.UserRepository,
	followRepo repository.FollowRepository,
	notifRepo repository.NotificationRepository,
	jwt *auth.JWTManager,
	log *slog.Logger,
) *Server {
	return &Server{
		userRepo:   userRepo,
		followRepo: followRepo,
		notifRepo:  notifRepo,
		jwt:        jwt,
		log:        log,
	}
}

func (s *Server) enrichUserWithCounts(ctx context.Context, user *pb.User) {
	if user == nil {
		return
	}
	followers, following, err := s.followRepo.GetFollowCounts(ctx, user.GetId())
	if err == nil {
		user.FollowersCount = followers
		user.FollowingCount = following
	}
}

func (s *Server) enrichUsersWithCounts(ctx context.Context, users []*pb.User) {
	if len(users) == 0 {
		return
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.GetId()
	}
	countsMap, err := s.followRepo.GetFollowCountsBatch(ctx, ids)
	if err != nil || countsMap == nil {
		return
	}
	for _, u := range users {
		if counts, ok := countsMap[u.GetId()]; ok {
			u.FollowersCount = counts[0]
			u.FollowingCount = counts[1]
		}
	}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("Register", "duration", time.Since(start)) }()

	if len(req.GetUsername()) < 3 || len(req.GetUsername()) > 50 {
		return nil, status.Error(codes.InvalidArgument, "username must be 3-50 characters")
	}
	if !isValidEmail(req.GetEmail()) {
		return nil, status.Error(codes.InvalidArgument, "invalid email format")
	}
	if len(req.GetPassword()) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), 12)
	if err != nil {
		s.log.Error("bcrypt hash failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	user, err := s.userRepo.Create(ctx, req.GetUsername(), req.GetEmail(), string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		if errors.Is(err, repository.ErrDuplicateUsername) {
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		}
		s.log.Error("create user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// First registered user becomes admin
	count, err := s.userRepo.CountUsers(ctx)
	if err == nil && count == 1 {
		if promoted, err := s.userRepo.SetAdmin(ctx, user.ID, true); err == nil {
			user = promoted
		}
	}

	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(ctx, user.ID, req.GetUaHash())
	if err != nil {
		s.log.Error("generate token pair failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.RegisterResponse{
		User:         toProtoUser(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("Login", "duration", time.Since(start)) }()

	user, err := s.userRepo.GetByEmail(ctx, req.GetEmail())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		s.log.Error("get user by email failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if user.IsBanned {
		return nil, status.Error(codes.PermissionDenied, "account is banned")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(ctx, user.ID, req.GetUaHash())
	if err != nil {
		s.log.Error("generate token pair failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	protoUser := toProtoUser(user)
	s.enrichUserWithCounts(ctx, protoUser)
	return &pb.LoginResponse{
		User:         protoUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("RefreshToken", "duration", time.Since(start)) }()

	accessToken, refreshToken, err := s.jwt.RotateRefreshToken(ctx, req.GetRefreshToken(), req.GetUaHash())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
		}
		s.log.Error("rotate refresh token failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("Logout", "duration", time.Since(start)) }()

	if err := s.jwt.Logout(ctx, req.GetAccessToken(), req.GetRefreshToken()); err != nil {
		s.log.Error("logout failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.LogoutResponse{}, nil
}

func (s *Server) RevokeAllTokens(ctx context.Context, req *pb.RevokeAllTokensRequest) (*pb.RevokeAllTokensResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("RevokeAllTokens", "duration", time.Since(start)) }()

	if err := s.jwt.RevokeAllTokens(ctx, req.GetUserId(), req.GetAccessToken()); err != nil {
		s.log.Error("revoke all tokens failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.RevokeAllTokensResponse{}, nil
}

func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetUser", "duration", time.Since(start)) }()

	user, err := s.userRepo.GetByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("get user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	protoUser := toProtoUser(user)
	s.enrichUserWithCounts(ctx, protoUser)

	var isFollowing bool
	if req.GetViewerId() != "" && req.GetViewerId() != req.GetId() {
		isFollowing, _ = s.followRepo.IsFollowing(ctx, req.GetViewerId(), req.GetId())
	}

	return &pb.GetUserResponse{User: protoUser, IsFollowing: isFollowing}, nil //nolint
}

func (s *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("UpdateUser", "duration", time.Since(start)) }()

	fields := make(map[string]string)
	if req.GetDisplayName() != "" {
		fields["display_name"] = req.GetDisplayName()
	}
	if req.GetBio() != "" {
		fields["bio"] = req.GetBio()
	}
	if req.GetAvatarUrl() != "" {
		fields["avatar_url"] = req.GetAvatarUrl()
	}
	if req.GetBannerUrl() != "" {
		fields["banner_url"] = req.GetBannerUrl()
	}

	user, err := s.userRepo.Update(ctx, req.GetId(), fields)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("update user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	protoUser := toProtoUser(user)
	s.enrichUserWithCounts(ctx, protoUser)
	return &pb.UpdateUserResponse{User: protoUser}, nil
}

func (s *Server) SoftDeleteUser(ctx context.Context, req *pb.SoftDeleteUserRequest) (*pb.SoftDeleteUserResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("SoftDeleteUser", "duration", time.Since(start)) }()

	if err := s.userRepo.SoftDelete(ctx, req.GetId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("soft delete user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.SoftDeleteUserResponse{}, nil
}

func (s *Server) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("SearchUsers", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	users, nextCursor, err := s.userRepo.Search(ctx, req.GetQuery(), cursor, int(limit))
	if err != nil {
		s.log.Error("search users failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		u := u
		pbUsers[i] = toProtoUser(&u)
	}

	s.enrichUsersWithCounts(ctx, pbUsers)

	return &pb.SearchUsersResponse{
		Users: pbUsers,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
		},
	}, nil
}

func (s *Server) GetUsersByIDs(ctx context.Context, req *pb.GetUsersByIDsRequest) (*pb.GetUsersByIDsResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetUsersByIDs", "duration", time.Since(start)) }()

	users, err := s.userRepo.GetByIDs(ctx, req.GetIds())
	if err != nil {
		s.log.Error("get users by ids failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		u := u
		pbUsers[i] = toProtoUser(&u)
	}

	s.enrichUsersWithCounts(ctx, pbUsers)

	return &pb.GetUsersByIDsResponse{Users: pbUsers}, nil
}

func (s *Server) Follow(ctx context.Context, req *pb.FollowRequest) (*pb.FollowResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("Follow", "duration", time.Since(start)) }()

	// Check target exists
	_, err := s.userRepo.GetByID(ctx, req.GetFollowingId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "target user not found")
		}
		s.log.Error("get target user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := s.followRepo.Follow(ctx, req.GetFollowerId(), req.GetFollowingId()); err != nil {
		if errors.Is(err, repository.ErrAlreadyFollowing) {
			return &pb.FollowResponse{}, nil
		}
		if errors.Is(err, repository.ErrSelfFollow) {
			return nil, status.Error(codes.InvalidArgument, "cannot follow yourself")
		}
		s.log.Error("follow failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.FollowResponse{}, nil
}

func (s *Server) Unfollow(ctx context.Context, req *pb.UnfollowRequest) (*pb.UnfollowResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("Unfollow", "duration", time.Since(start)) }()

	if err := s.followRepo.Unfollow(ctx, req.GetFollowerId(), req.GetFollowingId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "follow relationship not found")
		}
		s.log.Error("unfollow failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.UnfollowResponse{}, nil
}

func (s *Server) GetFollowers(ctx context.Context, req *pb.GetFollowersRequest) (*pb.GetFollowersResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetFollowers", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	ids, nextCursor, err := s.followRepo.GetFollowers(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get followers failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	users, err := s.userRepo.GetByIDs(ctx, ids)
	if err != nil {
		s.log.Error("get users by ids failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		u := u
		pbUsers[i] = toProtoUser(&u)
	}

	s.enrichUsersWithCounts(ctx, pbUsers)

	return &pb.GetFollowersResponse{
		Users: pbUsers,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
		},
	}, nil
}

func (s *Server) GetFollowing(ctx context.Context, req *pb.GetFollowingRequest) (*pb.GetFollowingResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("GetFollowing", "duration", time.Since(start)) }()

	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	ids, nextCursor, err := s.followRepo.GetFollowing(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get following failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	users, err := s.userRepo.GetByIDs(ctx, ids)
	if err != nil {
		s.log.Error("get users by ids failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		u := u
		pbUsers[i] = toProtoUser(&u)
	}

	s.enrichUsersWithCounts(ctx, pbUsers)

	return &pb.GetFollowingResponse{
		Users: pbUsers,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
		},
	}, nil
}

func (s *Server) GetRecommendedUsers(ctx context.Context, req *pb.GetRecommendedUsersRequest) (*pb.GetRecommendedUsersResponse, error) {
	users, err := s.userRepo.GetRecommended(ctx, req.GetUserId(), int(req.GetLimit()))
	if err != nil {
		s.log.Error("get recommended users failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		pbUsers[i] = toProtoUser(&u)
	}

	s.enrichUsersWithCounts(ctx, pbUsers)

	return &pb.GetRecommendedUsersResponse{Users: pbUsers}, nil
}

func (s *Server) CreateNotification(ctx context.Context, req *pb.CreateNotificationRequest) (*pb.CreateNotificationResponse, error) {
	// Don't notify about own actions
	if req.GetActorId() == req.GetUserId() {
		return &pb.CreateNotificationResponse{}, nil
	}

	var postID, commentID *string
	if req.GetPostId() != "" {
		pid := req.GetPostId()
		postID = &pid
	}
	if req.GetCommentId() != "" {
		cid := req.GetCommentId()
		commentID = &cid
	}

	n, err := s.notifRepo.Create(ctx, req.GetUserId(), req.GetActorId(), req.GetType(), postID, commentID)
	if err != nil {
		s.log.Error("create notification failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.CreateNotificationResponse{
		Notification: toProtoNotification(n),
	}, nil
}

func (s *Server) GetNotifications(ctx context.Context, req *pb.GetNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	notifications, nextCursor, hasMore, err := s.notifRepo.GetByUser(ctx, req.GetUserId(), cursor, int(limit))
	if err != nil {
		s.log.Error("get notifications failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Enrich with actor info
	actorIDs := make(map[string]struct{})
	for _, n := range notifications {
		actorIDs[n.ActorID] = struct{}{}
	}
	ids := make([]string, 0, len(actorIDs))
	for id := range actorIDs {
		ids = append(ids, id)
	}

	actorMap := make(map[string]*pb.User)
	if len(ids) > 0 {
		actors, err := s.userRepo.GetByIDs(ctx, ids)
		if err == nil {
			for _, a := range actors {
				actorMap[a.ID] = toProtoUser(&a)
			}
		}
	}

	pbNotifications := make([]*pb.Notification, len(notifications))
	for i, n := range notifications {
		pn := toProtoNotification(&n)
		pn.Actor = actorMap[n.ActorID]
		pbNotifications[i] = pn
	}

	return &pb.GetNotificationsResponse{
		Notifications: pbNotifications,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *Server) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	count, err := s.notifRepo.GetUnreadCount(ctx, req.GetUserId())
	if err != nil {
		s.log.Error("get unread count failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.GetUnreadCountResponse{Count: count}, nil
}

func (s *Server) MarkRead(ctx context.Context, req *pb.MarkReadRequest) (*pb.MarkReadResponse, error) {
	if err := s.notifRepo.MarkRead(ctx, req.GetId(), req.GetUserId()); err != nil {
		s.log.Error("mark read failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.MarkReadResponse{}, nil
}

func (s *Server) MarkAllRead(ctx context.Context, req *pb.MarkAllReadRequest) (*pb.MarkAllReadResponse, error) {
	if err := s.notifRepo.MarkAllRead(ctx, req.GetUserId()); err != nil {
		s.log.Error("mark all read failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.MarkAllReadResponse{}, nil
}

func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("ChangePassword", "duration", time.Since(start)) }()

	if len(req.GetNewPassword()) < 8 {
		return nil, status.Error(codes.InvalidArgument, "new password must be at least 8 characters")
	}

	user, err := s.userRepo.GetByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("get user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetCurrentPassword())); err != nil {
		return nil, status.Error(codes.Unauthenticated, "current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetNewPassword()), 12)
	if err != nil {
		s.log.Error("bcrypt hash failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	_, err = s.userRepo.Update(ctx, req.GetUserId(), map[string]string{"password_hash": string(hash)})
	if err != nil {
		s.log.Error("update password failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Revoke all other sessions after password change
	if err := s.jwt.RevokeAllTokens(ctx, req.GetUserId(), req.GetAccessToken()); err != nil {
		s.log.Error("revoke tokens after password change failed", "error", err)
	}

	return &pb.ChangePasswordResponse{}, nil
}

func (s *Server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	var cursor string
	var limit int32 = 20
	if req.GetPagination() != nil {
		cursor = req.GetPagination().GetCursor()
		if req.GetPagination().GetLimit() > 0 {
			limit = req.GetPagination().GetLimit()
		}
	}

	users, nextCursor, err := s.userRepo.ListAll(ctx, cursor, int(limit))
	if err != nil {
		s.log.Error("list users failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		u := u
		pbUsers[i] = toProtoUser(&u)
	}

	s.enrichUsersWithCounts(ctx, pbUsers)

	return &pb.ListUsersResponse{
		Users: pbUsers,
		Pagination: &commonpb.PaginationResponse{
			NextCursor: nextCursor,
			HasMore:    nextCursor != "",
		},
	}, nil
}

func (s *Server) SetAdmin(ctx context.Context, req *pb.SetAdminRequest) (*pb.SetAdminResponse, error) {
	// Prevent removing the last admin
	if !req.GetIsAdmin() {
		count, err := s.userRepo.CountAdmins(ctx)
		if err != nil {
			s.log.Error("count admins failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		if count <= 1 {
			return nil, status.Error(codes.FailedPrecondition, "cannot remove the last admin")
		}
	}

	user, err := s.userRepo.SetAdmin(ctx, req.GetUserId(), req.GetIsAdmin())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("set admin failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	protoUser := toProtoUser(user)
	s.enrichUserWithCounts(ctx, protoUser)
	return &pb.SetAdminResponse{User: protoUser}, nil
}

func (s *Server) BanUser(ctx context.Context, req *pb.BanUserRequest) (*pb.BanUserResponse, error) {
	user, err := s.userRepo.SetBanned(ctx, req.GetUserId(), req.GetIsBanned())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("ban user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Revoke all tokens when banning to force immediate logout
	if req.GetIsBanned() {
		_ = s.jwt.RevokeAllTokens(ctx, req.GetUserId(), "")
	}

	protoUser := toProtoUser(user)
	s.enrichUserWithCounts(ctx, protoUser)
	return &pb.BanUserResponse{User: protoUser}, nil
}

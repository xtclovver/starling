package handler

import (
	"encoding/json"
	"net/http"

	"github.com/usedcvnt/microtwitter/api-gateway/internal/middleware"
	commentpb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type response struct {
	Data  any    `json:"data"`
	Error *errBody `json:"error"`
}

type errBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{Data: data})
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{Error: &errBody{Code: code, Message: msg}})
}

func handleGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	switch st.Code() {
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, st.Message())
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func getUserID(r *http.Request) string {
	id, _ := r.Context().Value(middleware.UserIDKey).(string)
	return id
}

func tsToString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format("2006-01-02T15:04:05Z")
}

func postToMap(p *postpb.Post) map[string]any {
	m := map[string]any{
		"id":             p.GetId(),
		"user_id":        p.GetUserId(),
		"content":        p.GetContent(),
		"media_url":      p.GetMediaUrl(),
		"likes_count":    p.GetLikesCount(),
		"comments_count": p.GetCommentsCount(),
		"reposts_count":  p.GetRepostsCount(),
		"created_at":     tsToString(p.GetCreatedAt()),
		"updated_at":     tsToString(p.GetUpdatedAt()),
		"bookmarked":     p.GetBookmarked(),
		"reposted":       p.GetReposted(),
		"liked":          p.GetLiked(),
		"hashtags":       p.GetHashtags(),
	}
	if p.GetEditedAt() != nil {
		m["edited_at"] = tsToString(p.GetEditedAt())
	}
	return m
}

func userToMap(u *userpb.User) map[string]any {
	return map[string]any{
		"id":              u.GetId(),
		"username":        u.GetUsername(),
		"email":           u.GetEmail(),
		"display_name":    u.GetDisplayName(),
		"bio":             u.GetBio(),
		"avatar_url":      u.GetAvatarUrl(),
		"created_at":      tsToString(u.GetCreatedAt()),
		"updated_at":      tsToString(u.GetUpdatedAt()),
		"followers_count": u.GetFollowersCount(),
		"following_count": u.GetFollowingCount(),
	}
}

func notificationToMap(n *userpb.Notification) map[string]any {
	m := map[string]any{
		"id":         n.GetId(),
		"user_id":    n.GetUserId(),
		"actor_id":   n.GetActorId(),
		"type":       n.GetType(),
		"post_id":    n.GetPostId(),
		"comment_id": n.GetCommentId(),
		"read":       n.GetRead(),
		"created_at": tsToString(n.GetCreatedAt()),
	}
	if n.GetActor() != nil {
		m["actor"] = userToMap(n.GetActor())
	}
	return m
}

func commentToMap(c *commentpb.Comment) map[string]any {
	children := make([]map[string]any, len(c.GetChildren()))
	for i, ch := range c.GetChildren() {
		children[i] = commentToMap(ch)
	}
	return map[string]any{
		"id":          c.GetId(),
		"post_id":     c.GetPostId(),
		"user_id":     c.GetUserId(),
		"parent_id":   c.GetParentId(),
		"content":     c.GetContent(),
		"likes_count": c.GetLikesCount(),
		"liked":       c.GetLiked(),
		"depth":       c.GetDepth(),
		"created_at":  tsToString(c.GetCreatedAt()),
		"updated_at":  tsToString(c.GetUpdatedAt()),
		"children":    children,
	}
}

func usersToList(users []*userpb.User) []map[string]any {
	result := make([]map[string]any, len(users))
	for i, u := range users {
		result[i] = userToMap(u)
	}
	return result
}

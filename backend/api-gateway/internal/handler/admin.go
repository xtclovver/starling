package handler

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	commentpb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

type AdminHandler struct {
	user    userpb.UserServiceClient
	post    postpb.PostServiceClient
	comment commentpb.CommentServiceClient
}

func NewAdminHandler(user userpb.UserServiceClient, post postpb.PostServiceClient, comment commentpb.CommentServiceClient) *AdminHandler {
	return &AdminHandler{user: user, post: post, comment: comment}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := int32(20)

	resp, err := h.user.ListUsers(r.Context(), &userpb.ListUsersRequest{
		Pagination: &commonpb.PaginationRequest{
			Cursor: cursor,
			Limit:  limit,
		},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	users := usersToList(resp.GetUsers())
	writeJSON(w, http.StatusOK, map[string]any{
		"users":      users,
		"pagination": paginationToMap(resp.GetPagination()),
	})
}

func (h *AdminHandler) SetAdmin(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	var req struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.user.SetAdmin(r.Context(), &userpb.SetAdminRequest{
		UserId:  userID,
		IsAdmin: req.IsAdmin,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userToMap(resp.GetUser()))
}

func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	var req struct {
		IsBanned bool `json:"is_banned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.user.BanUser(r.Context(), &userpb.BanUserRequest{
		UserId:   userID,
		IsBanned: req.IsBanned,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, userToMap(resp.GetUser()))
}

func (h *AdminHandler) AdminDeletePost(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")

	_, err := h.post.DeletePost(r.Context(), &postpb.DeletePostRequest{
		Id:      postID,
		UserId:  getUserID(r),
		IsAdmin: true,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *AdminHandler) AdminDeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")

	_, err := h.comment.DeleteComment(r.Context(), &commentpb.DeleteCommentRequest{
		Id:      commentID,
		UserId:  getUserID(r),
		IsAdmin: true,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func paginationToMap(p *commonpb.PaginationResponse) map[string]any {
	if p == nil {
		return map[string]any{"next_cursor": "", "has_more": false}
	}
	return map[string]any{
		"next_cursor": p.GetNextCursor(),
		"has_more":    p.GetHasMore(),
	}
}

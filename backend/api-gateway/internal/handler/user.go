package handler

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

type UserHandler struct {
	user userpb.UserServiceClient
	post postpb.PostServiceClient
}

func NewUserHandler(user userpb.UserServiceClient, post postpb.PostServiceClient) *UserHandler {
	return &UserHandler{user: user, post: post}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	viewerID := getUserID(r)
	resp, err := h.user.GetUser(r.Context(), &userpb.GetUserRequest{Id: id, ViewerId: viewerID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	u := userToMap(resp.GetUser())
	u["is_following"] = resp.GetIsFollowing()
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

type updateUserRequest struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)
	if userID != id {
		writeError(w, http.StatusForbidden, "cannot update another user")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.user.UpdateUser(r.Context(), &userpb.UpdateUserRequest{
		Id:          id,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarUrl:   req.AvatarURL,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": userToMap(resp.GetUser())})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)
	if userID != id {
		writeError(w, http.StatusForbidden, "cannot delete another user")
		return
	}

	_, err := h.user.SoftDeleteUser(r.Context(), &userpb.SoftDeleteUserRequest{Id: id})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")
	viewerID := getUserID(r)

	resp, err := h.post.GetPostsByUser(r.Context(), &postpb.GetPostsByUserRequest{
		UserId:     id,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
		ViewerId:   viewerID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	posts := resp.GetPosts()
	userResp, _ := h.user.GetUser(r.Context(), &userpb.GetUserRequest{Id: id})
	var author map[string]any
	if userResp != nil && userResp.GetUser() != nil {
		author = userToMap(userResp.GetUser())
	}

	enriched := make([]map[string]any, len(posts))
	for i, p := range posts {
		m := postToMap(p)
		if author != nil {
			m["author"] = author
		}
		enriched[i] = m
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"posts":      enriched,
		"pagination": resp.GetPagination(),
	})
}

func (h *UserHandler) GetUserReposts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")
	viewerID := getUserID(r)

	resp, err := h.post.GetRepostsByUser(r.Context(), &postpb.GetRepostsByUserRequest{
		UserId:     id,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
		ViewerId:   viewerID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	posts := resp.GetPosts()

	// Batch-fetch authors
	userMap := make(map[string]map[string]any)
	if len(posts) > 0 {
		seen := make(map[string]struct{})
		var ids []string
		for _, p := range posts {
			if _, ok := seen[p.GetUserId()]; !ok {
				seen[p.GetUserId()] = struct{}{}
				ids = append(ids, p.GetUserId())
			}
		}
		usersResp, _ := h.user.GetUsersByIDs(r.Context(), &userpb.GetUsersByIDsRequest{Ids: ids})
		if usersResp != nil {
			for _, u := range usersResp.GetUsers() {
				userMap[u.GetId()] = userToMap(u)
			}
		}
	}

	enriched := make([]map[string]any, len(posts))
	for i, p := range posts {
		m := postToMap(p)
		if author, ok := userMap[p.GetUserId()]; ok {
			m["author"] = author
		}
		enriched[i] = m
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"posts":      enriched,
		"pagination": resp.GetPagination(),
	})
}

func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.user.SearchUsers(r.Context(), &userpb.SearchUsersRequest{
		Query:      q,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	users := make([]map[string]any, len(resp.GetUsers()))
	for i, u := range resp.GetUsers() {
		users[i] = userToMap(u)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users":      users,
		"pagination": resp.GetPagination(),
	})
}

func (h *UserHandler) Follow(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.user.Follow(r.Context(), &userpb.FollowRequest{
		FollowerId:  userID,
		FollowingId: targetID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.user.Unfollow(r.Context(), &userpb.UnfollowRequest{
		FollowerId:  userID,
		FollowingId: targetID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.user.GetFollowers(r.Context(), &userpb.GetFollowersRequest{
		UserId:     id,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      usersToList(resp.GetUsers()),
		"pagination": resp.GetPagination(),
	})
}

func (h *UserHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.user.GetFollowing(r.Context(), &userpb.GetFollowingRequest{
		UserId:     id,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      usersToList(resp.GetUsers()),
		"pagination": resp.GetPagination(),
	})
}

func (h *UserHandler) GetRecommendedUsers(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	resp, err := h.user.GetRecommendedUsers(r.Context(), &userpb.GetRecommendedUsersRequest{
		UserId: userID,
		Limit:  5,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"users": usersToList(resp.GetUsers())})
}

func (h *UserHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.user.GetNotifications(r.Context(), &userpb.GetNotificationsRequest{
		UserId:     userID,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	notifs := make([]map[string]any, len(resp.GetNotifications()))
	for i, n := range resp.GetNotifications() {
		notifs[i] = notificationToMap(n)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": notifs,
		"pagination":    resp.GetPagination(),
	})
}

func (h *UserHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	resp, err := h.user.GetUnreadCount(r.Context(), &userpb.GetUnreadCountRequest{UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"count": resp.GetCount()})
}

func (h *UserHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.user.MarkRead(r.Context(), &userpb.MarkReadRequest{Id: id, UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *UserHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	_, err := h.user.MarkAllRead(r.Context(), &userpb.MarkAllReadRequest{UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

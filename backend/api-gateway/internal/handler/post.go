package handler

import (
	"encoding/json"
	"net/http"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

type PostHandler struct {
	post postpb.PostServiceClient
	user userpb.UserServiceClient
}

func NewPostHandler(post postpb.PostServiceClient, user userpb.UserServiceClient) *PostHandler {
	return &PostHandler{post: post, user: user}
}

type createPostRequest struct {
	Content  string `json:"content"`
	MediaURL string `json:"media_url,omitempty"`
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := getUserID(r)
	resp, err := h.post.CreatePost(r.Context(), &postpb.CreatePostRequest{
		UserId:   userID,
		Content:  req.Content,
		MediaUrl: req.MediaURL,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp.GetPost())
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	resp, err := h.post.GetPost(r.Context(), &postpb.GetPostRequest{Id: id})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp.GetPost())
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.DeletePost(r.Context(), &postpb.DeletePostRequest{
		Id:     id,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *PostHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.post.GetFeed(r.Context(), &postpb.GetFeedRequest{
		UserId:     userID,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	// Batch enrich authors
	posts := resp.GetPosts()
	if len(posts) > 0 {
		userIDs := make(map[string]struct{})
		for _, p := range posts {
			userIDs[p.GetUserId()] = struct{}{}
		}
		ids := make([]string, 0, len(userIDs))
		for id := range userIDs {
			ids = append(ids, id)
		}
		usersResp, _ := h.user.GetUsersByIDs(r.Context(), &userpb.GetUsersByIDsRequest{Ids: ids})

		userMap := make(map[string]*userpb.User)
		if usersResp != nil {
			for _, u := range usersResp.GetUsers() {
				userMap[u.GetId()] = u
			}
		}

		type enrichedPost struct {
			Post   any `json:"post"`
			Author any `json:"author,omitempty"`
		}

		enriched := make([]enrichedPost, len(posts))
		for i, p := range posts {
			enriched[i] = enrichedPost{Post: p, Author: userMap[p.GetUserId()]}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"posts":      enriched,
			"pagination": resp.GetPagination(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"posts":      posts,
		"pagination": resp.GetPagination(),
	})
}

func (h *PostHandler) LikePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.LikePost(r.Context(), &postpb.LikePostRequest{
		PostId: id,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *PostHandler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.UnlikePost(r.Context(), &postpb.UnlikePostRequest{
		PostId: id,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

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

	post := postToMap(resp.GetPost())
	h.enrichSinglePost(r, post)
	writeJSON(w, http.StatusCreated, map[string]any{"post": post})
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	viewerID := getUserID(r)
	resp, err := h.post.GetPost(r.Context(), &postpb.GetPostRequest{Id: id, UserId: viewerID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	post := postToMap(resp.GetPost())
	h.enrichSinglePost(r, post)
	writeJSON(w, http.StatusOK, map[string]any{"post": post})
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

	posts := resp.GetPosts()
	h.enrichPosts(r, w, posts, resp.GetPagination())
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

func (h *PostHandler) GetGlobalFeed(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	viewerID := getUserID(r)

	resp, err := h.post.GetGlobalFeed(r.Context(), &postpb.GetGlobalFeedRequest{
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
		UserId:     viewerID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	posts := resp.GetPosts()
	h.enrichPosts(r, w, posts, resp.GetPagination())
}

func (h *PostHandler) BookmarkPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.BookmarkPost(r.Context(), &postpb.BookmarkPostRequest{PostId: id, UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *PostHandler) UnbookmarkPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.UnbookmarkPost(r.Context(), &postpb.UnbookmarkPostRequest{PostId: id, UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *PostHandler) GetBookmarks(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.post.GetBookmarks(r.Context(), &postpb.GetBookmarksRequest{
		UserId:     userID,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	posts := resp.GetPosts()
	h.enrichPosts(r, w, posts, resp.GetPagination())
}

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	var req struct {
		Content  string `json:"content"`
		MediaURL string `json:"media_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.post.UpdatePost(r.Context(), &postpb.UpdatePostRequest{
		Id:       id,
		UserId:   userID,
		Content:  req.Content,
		MediaUrl: req.MediaURL,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	post := postToMap(resp.GetPost())
	h.enrichSinglePost(r, post)
	writeJSON(w, http.StatusOK, map[string]any{"post": post})
}

func (h *PostHandler) GetPostsByHashtag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	cursor := r.URL.Query().Get("cursor")
	viewerID := getUserID(r)

	resp, err := h.post.GetPostsByHashtag(r.Context(), &postpb.GetPostsByHashtagRequest{
		Tag:        tag,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
		UserId:     viewerID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	posts := resp.GetPosts()
	h.enrichPosts(r, w, posts, resp.GetPagination())
}

func (h *PostHandler) GetTrendingHashtags(w http.ResponseWriter, r *http.Request) {
	resp, err := h.post.GetTrendingHashtags(r.Context(), &postpb.GetTrendingHashtagsRequest{Limit: 10})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	hashtags := make([]map[string]any, len(resp.GetHashtags()))
	for i, h := range resp.GetHashtags() {
		hashtags[i] = map[string]any{
			"tag":        h.GetTag(),
			"post_count": h.GetPostCount(),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"hashtags": hashtags})
}

func (h *PostHandler) RepostPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.RepostPost(r.Context(), &postpb.RepostPostRequest{PostId: id, UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *PostHandler) UnrepostPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.UnrepostPost(r.Context(), &postpb.UnrepostPostRequest{PostId: id, UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *PostHandler) QuotePost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.post.QuotePost(r.Context(), &postpb.QuotePostRequest{
		PostId:  id,
		UserId:  userID,
		Content: req.Content,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	post := postToMap(resp.GetPost())
	h.enrichSinglePost(r, post)
	writeJSON(w, http.StatusCreated, map[string]any{"post": post})
}

func (h *PostHandler) enrichSinglePost(r *http.Request, post map[string]any) {
	userID, _ := post["user_id"].(string)
	if userID == "" {
		return
	}
	usersResp, err := h.user.GetUsersByIDs(r.Context(), &userpb.GetUsersByIDsRequest{Ids: []string{userID}})
	if err != nil || usersResp == nil {
		return
	}
	for _, u := range usersResp.GetUsers() {
		if u.GetId() == userID {
			post["author"] = userToMap(u)
			return
		}
	}
}

// enrichPosts batch-enriches posts with author info and writes the response.
func (h *PostHandler) enrichPosts(r *http.Request, w http.ResponseWriter, posts []*postpb.Post, pagination *commonpb.PaginationResponse) {
	enriched := make([]map[string]any, len(posts))
	userMap := make(map[string]*userpb.User)

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
		if usersResp != nil {
			for _, u := range usersResp.GetUsers() {
				userMap[u.GetId()] = u
			}
		}
	}

	for i, p := range posts {
		m := postToMap(p)
		if u, ok := userMap[p.GetUserId()]; ok {
			m["author"] = userToMap(u)
		}
		enriched[i] = m
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"posts":      enriched,
		"pagination": pagination,
	})
}

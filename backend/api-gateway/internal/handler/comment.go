package handler

import (
	"context"
	"encoding/json"
	"net/http"

	commentpb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

type CommentHandler struct {
	comment  commentpb.CommentServiceClient
	user     userpb.UserServiceClient
	post     postpb.PostServiceClient
	notifier Notifier
}

func NewCommentHandler(comment commentpb.CommentServiceClient, user userpb.UserServiceClient, post postpb.PostServiceClient, notifier Notifier) *CommentHandler {
	return &CommentHandler{comment: comment, user: user, post: post, notifier: notifier}
}

type createCommentRequest struct {
	ParentID string `json:"parent_id,omitempty"`
	Content  string `json:"content"`
	MediaURL string `json:"media_url,omitempty"`
}

func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	userID := getUserID(r)

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.comment.CreateComment(r.Context(), &commentpb.CreateCommentRequest{
		PostId:   postID,
		UserId:   userID,
		ParentId: req.ParentID,
		Content:  req.Content,
		MediaUrl: req.MediaURL,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	comment := resp.GetComment()
	cm := commentToMap(comment)
	if userID != "" {
		usersResp, _ := h.user.GetUsersByIDs(r.Context(), &userpb.GetUsersByIDsRequest{Ids: []string{userID}})
		if usersResp != nil {
			for _, u := range usersResp.GetUsers() {
				if u.GetId() == userID {
					cm["author"] = userToMap(u)
				}
			}
		}
	}

	// Notify post owner async
	go func() {
		postResp, err := h.post.GetPost(context.Background(), &postpb.GetPostRequest{Id: postID})
		if err != nil || postResp.GetPost().GetUserId() == userID {
			return
		}
		ownerID := postResp.GetPost().GetUserId()
		nr, err := h.user.CreateNotification(context.Background(), &userpb.CreateNotificationRequest{
			UserId:    ownerID,
			ActorId:   userID,
			Type:      "new_comment",
			PostId:    postID,
			CommentId: comment.GetId(),
		})
		if err == nil && h.notifier != nil {
			h.notifier.PublishNotification(context.Background(), ownerID, notificationToMap(nr.GetNotification()))
		}
	}()

	go notifyMentions(context.Background(), req.Content, userID, postID, comment.GetId(), h.user, h.notifier)

	writeJSON(w, http.StatusCreated, map[string]any{"comment": cm})
}

func (h *CommentHandler) GetCommentTree(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")
	userID := getUserID(r)

	resp, err := h.comment.GetCommentTree(r.Context(), &commentpb.GetCommentTreeRequest{
		PostId:     postID,
		UserId:     userID,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	comments := make([]map[string]any, len(resp.GetComments()))
	for i, c := range resp.GetComments() {
		comments[i] = commentToMap(c)
	}

	userIDs := collectUserIDsFromComments(comments)
	if len(userIDs) > 0 {
		usersResp, err := h.user.GetUsersByIDs(r.Context(), &userpb.GetUsersByIDsRequest{Ids: userIDs})
		if err == nil && usersResp != nil {
			userMap := make(map[string]map[string]any)
			for _, u := range usersResp.GetUsers() {
				userMap[u.GetId()] = userToMap(u)
			}
			enrichCommentsWithAuthors(comments, userMap)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"comments":   comments,
		"pagination": resp.GetPagination(),
	})
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.comment.DeleteComment(r.Context(), &commentpb.DeleteCommentRequest{
		Id:     commentID,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

type updateCommentRequest struct {
	Content  string `json:"content"`
	MediaURL string `json:"media_url,omitempty"`
}

func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	userID := getUserID(r)

	var req updateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.comment.UpdateComment(r.Context(), &commentpb.UpdateCommentRequest{
		Id:       commentID,
		UserId:   userID,
		Content:  req.Content,
		MediaUrl: req.MediaURL,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"comment": commentToMap(resp.GetComment())})
}

func (h *CommentHandler) LikeComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.comment.LikeComment(r.Context(), &commentpb.LikeCommentRequest{
		CommentId: commentID,
		UserId:    userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *CommentHandler) UnlikeComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.comment.UnlikeComment(r.Context(), &commentpb.UnlikeCommentRequest{
		CommentId: commentID,
		UserId:    userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func collectUserIDsFromComments(comments []map[string]any) []string {
	seen := make(map[string]struct{})
	var collect func([]map[string]any)
	collect = func(cs []map[string]any) {
		for _, c := range cs {
			if uid, ok := c["user_id"].(string); ok && uid != "" {
				seen[uid] = struct{}{}
			}
			if children, ok := c["children"].([]map[string]any); ok {
				collect(children)
			}
		}
	}
	collect(comments)
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

func enrichCommentsWithAuthors(comments []map[string]any, userMap map[string]map[string]any) {
	for _, c := range comments {
		if uid, ok := c["user_id"].(string); ok {
			if author, ok := userMap[uid]; ok {
				c["author"] = author
			}
		}
		if children, ok := c["children"].([]map[string]any); ok {
			enrichCommentsWithAuthors(children, userMap)
		}
	}
}

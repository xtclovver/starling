package handler

import (
	"encoding/json"
	"net/http"

	commentpb "github.com/usedcvnt/microtwitter/gen/go/comment/v1"
	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
)

type CommentHandler struct {
	comment commentpb.CommentServiceClient
}

func NewCommentHandler(comment commentpb.CommentServiceClient) *CommentHandler {
	return &CommentHandler{comment: comment}
}

type createCommentRequest struct {
	ParentID string `json:"parent_id,omitempty"`
	Content  string `json:"content"`
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
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp.GetComment())
}

func (h *CommentHandler) GetCommentTree(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")

	resp, err := h.comment.GetCommentTree(r.Context(), &commentpb.GetCommentTreeRequest{
		PostId:     postID,
		Pagination: &commonpb.PaginationRequest{Cursor: cursor, Limit: 20},
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"comments":   resp.GetComments(),
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

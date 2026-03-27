package handler

import (
	"io"
	"net/http"

	mediapb "github.com/usedcvnt/microtwitter/gen/go/media/v1"
)

type MediaHandler struct {
	media mediapb.MediaServiceClient
}

func NewMediaHandler(media mediapb.MediaServiceClient) *MediaHandler {
	return &MediaHandler{media: media}
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 10MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	resp, err := h.media.UploadMedia(r.Context(), &mediapb.UploadMediaRequest{
		UserId:      userID,
		PostId:      r.FormValue("post_id"),
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	m := resp.GetMedia()
	writeJSON(w, http.StatusCreated, map[string]any{
		"media": map[string]any{
			"id":  m.GetId(),
			"url": m.GetUrl(),
		},
	})
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "media id is required")
		return
	}

	_, err := h.media.DeleteMedia(r.Context(), &mediapb.DeleteMediaRequest{
		Id:     id,
		UserId: userID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

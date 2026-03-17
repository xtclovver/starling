package handler

import (
	"encoding/json"
	"net/http"

	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

type AuthHandler struct {
	user userpb.UserServiceClient
}

func NewAuthHandler(user userpb.UserServiceClient) *AuthHandler {
	return &AuthHandler{user: user}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.user.Register(r.Context(), &userpb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":          resp.GetUser(),
		"access_token":  resp.GetAccessToken(),
		"refresh_token": resp.GetRefreshToken(),
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.user.Login(r.Context(), &userpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":          resp.GetUser(),
		"access_token":  resp.GetAccessToken(),
		"refresh_token": resp.GetRefreshToken(),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.user.RefreshToken(r.Context(), &userpb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  resp.GetAccessToken(),
		"refresh_token": resp.GetRefreshToken(),
	})
}

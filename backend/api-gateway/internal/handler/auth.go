package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

type AuthHandler struct {
	user       userpb.UserServiceClient
	cookiePath string
	secure     bool
}

func NewAuthHandler(user userpb.UserServiceClient, secureCookie bool) *AuthHandler {
	return &AuthHandler{user: user, cookiePath: "/api/auth", secure: secureCookie}
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 3600,
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
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

	h.setRefreshCookie(w, resp.GetRefreshToken())
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":         userToMap(resp.GetUser()),
		"access_token": resp.GetAccessToken(),
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

	h.setRefreshCookie(w, resp.GetRefreshToken())
	writeJSON(w, http.StatusOK, map[string]any{
		"user":         userToMap(resp.GetUser()),
		"access_token": resp.GetAccessToken(),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	resp, err := h.user.RefreshToken(r.Context(), &userpb.RefreshTokenRequest{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	h.setRefreshCookie(w, resp.GetRefreshToken())
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": resp.GetAccessToken(),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	accessToken := extractBearerToken(r)
	var refreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	_, _ = h.user.Logout(r.Context(), &userpb.LogoutRequest{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, nil)
}

func (h *AuthHandler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	accessToken := extractBearerToken(r)
	_, _ = h.user.RevokeAllTokens(r.Context(), &userpb.RevokeAllTokensRequest{
		UserId:      userID,
		AccessToken: accessToken,
	})

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, nil)
}


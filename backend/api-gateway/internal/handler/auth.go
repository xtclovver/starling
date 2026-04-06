package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
	"google.golang.org/grpc/metadata"
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

// computeUAHash returns the first 16 hex chars of SHA256(User-Agent).
func computeUAHash(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ua))
	return hex.EncodeToString(sum[:])[:16]
}

func clientMetadataCtx(r *http.Request) context.Context {
	ip := realClientIP(r)
	ua := r.Header.Get("User-Agent")
	md := metadata.Pairs("x-client-ip", ip, "x-client-ua", ua)
	return metadata.NewOutgoingContext(r.Context(), md)
}

// realClientIP extracts the client IP from RemoteAddr only.
// X-Forwarded-For is intentionally ignored: it is user-controlled and
// cannot be trusted without a verified list of trusted proxy CIDRs.
func realClientIP(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
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

	resp, err := h.user.Register(clientMetadataCtx(r), &userpb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		UaHash:   computeUAHash(r),
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

	resp, err := h.user.Login(clientMetadataCtx(r), &userpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		UaHash:   computeUAHash(r),
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
		UaHash:       computeUAHash(r),
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

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	accessToken := extractBearerToken(r)

	_, err := h.user.ChangePassword(r.Context(), &userpb.ChangePasswordRequest{
		UserId:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		AccessToken:     accessToken,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, nil)
}


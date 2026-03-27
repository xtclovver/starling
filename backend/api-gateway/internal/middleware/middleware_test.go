package middleware

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-jwt-secret-key"

func generateTestToken(secret string, userID string, exp time.Time) string {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": exp.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

// --- Auth middleware tests ---

func TestAuthRequired_ValidToken(t *testing.T) {
	auth := NewAuth(testSecret, nil)
	token := generateTestToken(testSecret, "user-123", time.Now().Add(time.Hour))

	var capturedUserID string
	handler := auth.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID, _ = r.Context().Value(UserIDKey).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedUserID != "user-123" {
		t.Errorf("expected user_id 'user-123', got %q", capturedUserID)
	}
}

func TestAuthRequired_NoToken(t *testing.T) {
	auth := NewAuth(testSecret, nil)

	handler := auth.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	auth := NewAuth(testSecret, nil)

	handler := auth.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthRequired_ExpiredToken(t *testing.T) {
	auth := NewAuth(testSecret, nil)
	token := generateTestToken(testSecret, "user-123", time.Now().Add(-time.Hour))

	handler := auth.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthRequired_WrongSecret(t *testing.T) {
	auth := NewAuth(testSecret, nil)
	token := generateTestToken("wrong-secret", "user-123", time.Now().Add(time.Hour))

	handler := auth.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthOptional_WithToken(t *testing.T) {
	auth := NewAuth(testSecret, nil)
	token := generateTestToken(testSecret, "user-456", time.Now().Add(time.Hour))

	var capturedUserID string
	handler := auth.Optional(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID, _ = r.Context().Value(UserIDKey).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedUserID != "user-456" {
		t.Errorf("expected user_id 'user-456', got %q", capturedUserID)
	}
}

func TestAuthOptional_WithoutToken(t *testing.T) {
	auth := NewAuth(testSecret, nil)

	var called bool
	var capturedUserID string
	handler := auth.Optional(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		capturedUserID, _ = r.Context().Value(UserIDKey).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should be called even without token")
	}
	if capturedUserID != "" {
		t.Errorf("expected empty user_id, got %q", capturedUserID)
	}
}

// --- Recovery middleware tests ---

func TestRecovery_NoPanic(t *testing.T) {
	log := slog.Default()
	handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRecovery_WithPanic(t *testing.T) {
	log := slog.Default()
	handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// --- SecurityHeaders middleware tests ---

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Content-Security-Policy", "default-src 'self'"},
	}

	for _, tt := range tests {
		got := rr.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s: expected %q, got %q", tt.header, tt.want, got)
		}
	}
}

// --- CORS middleware tests ---

func TestCORS_Preflight(t *testing.T) {
	handler := CORS("http://localhost:3000")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/posts", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected CORS origin 'http://localhost:3000', got %q", got)
	}
}

func TestCORS_RegularRequest(t *testing.T) {
	handler := CORS("http://localhost:3000")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected CORS origin, got %q", got)
	}
}

// --- BodyLimit middleware tests ---

func TestBodyLimit_WithinLimit(t *testing.T) {
	handler := BodyLimit(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 200)
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() == "http: request body too large" {
			t.Error("should not exceed body limit")
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.NewReader(make([]byte, 50))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestBodyLimit_ExceedsLimit(t *testing.T) {
	handler := BodyLimit(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 200)
		_, err := r.Body.Read(buf)
		if err == nil {
			t.Error("expected error reading body beyond limit")
		}
	}))

	body := bytes.NewReader(make([]byte, 200))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
}

// --- RequestID middleware tests ---

func TestRequestID_AddsHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	// Verify UUID-like format (36 chars with dashes)
	if len(requestID) != 36 {
		t.Errorf("expected UUID format (36 chars), got %d chars: %s", len(requestID), requestID)
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d", i), nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		id := rr.Header().Get("X-Request-ID")
		if ids[id] {
			t.Errorf("duplicate request ID: %s", id)
		}
		ids[id] = true
	}
}

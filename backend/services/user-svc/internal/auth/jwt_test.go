package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func setupMiniredis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestGenerateTokenPair(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	access, refresh, err := m.GenerateTokenPair(ctx, "user-123", "ua1")
	if err != nil {
		t.Fatalf("GenerateTokenPair error: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}

	userID, jti, err := m.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("got userID %q, want %q", userID, "user-123")
	}
	if jti == "" {
		t.Error("jti should not be empty")
	}
}

func TestRotateRefreshToken_Success(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	_, refresh, err := m.GenerateTokenPair(ctx, "user-rotate", "ua1")
	if err != nil {
		t.Fatalf("GenerateTokenPair error: %v", err)
	}

	newAccess, newRefresh, err := m.RotateRefreshToken(ctx, refresh, "ua1")
	if err != nil {
		t.Fatalf("RotateRefreshToken error: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Fatal("expected non-empty new tokens")
	}

	userID, _, err := m.ValidateAccessToken(newAccess)
	if err != nil {
		t.Fatalf("ValidateAccessToken on rotated token: %v", err)
	}
	if userID != "user-rotate" {
		t.Errorf("got userID %q, want %q", userID, "user-rotate")
	}
}

func TestRotateRefreshToken_ReuseDetection(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	_, refresh, _ := m.GenerateTokenPair(ctx, "user-reuse", "ua1")

	// First rotation succeeds and yields new token
	_, newRefresh, err := m.RotateRefreshToken(ctx, refresh, "ua1")
	if err != nil {
		t.Fatalf("first rotation failed: %v", err)
	}

	// Re-using the original token must be rejected
	_, _, err = m.RotateRefreshToken(ctx, refresh, "ua1")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken on reuse, got %v", err)
	}

	// After reuse detection the new token from the same family must also be revoked
	_, _, err = m.RotateRefreshToken(ctx, newRefresh, "ua1")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for sibling token after reuse, got %v", err)
	}
}

func TestRotateRefreshToken_UAMismatch(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	_, refresh, _ := m.GenerateTokenPair(ctx, "user-ua", "original-ua")

	_, _, err := m.RotateRefreshToken(ctx, refresh, "different-ua")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken on UA mismatch, got %v", err)
	}

	// Family must be invalidated — original token also gone
	_, _, err = m.RotateRefreshToken(ctx, refresh, "original-ua")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken after family invalidation, got %v", err)
	}
}

func TestRotateRefreshToken_UnknownToken(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	_, _, err := m.RotateRefreshToken(ctx, "totally-bogus-token", "ua1")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestLogout(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	access, refresh, _ := m.GenerateTokenPair(ctx, "user-logout", "ua1")

	if err := m.Logout(ctx, access, refresh); err != nil {
		t.Fatalf("Logout error: %v", err)
	}

	// refresh token must be invalid after logout
	_, _, err := m.RotateRefreshToken(ctx, refresh, "ua1")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken after logout, got %v", err)
	}
}

func TestRevokeAllTokens(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)
	ctx := context.Background()

	access, refresh1, _ := m.GenerateTokenPair(ctx, "user-revoke", "ua1")
	_, refresh2, _ := m.GenerateTokenPair(ctx, "user-revoke", "ua2")

	if err := m.RevokeAllTokens(ctx, "user-revoke", access); err != nil {
		t.Fatalf("RevokeAllTokens error: %v", err)
	}

	for _, rt := range []string{refresh1, refresh2} {
		_, _, err := m.RotateRefreshToken(ctx, rt, "ua1")
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken after revoke-all, got %v", err)
		}
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user-789",
		"iat": now.Add(-2 * time.Hour).Unix(),
		"exp": now.Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("test-secret"))

	_, _, err := m.ValidateAccessToken(tokenStr)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	for _, tc := range []string{"", "not-a-jwt", "a.b.c"} {
		_, _, err := m.ValidateAccessToken(tc)
		if err != ErrInvalidToken {
			t.Errorf("token %q: expected ErrInvalidToken, got %v", tc, err)
		}
	}
}

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
	return redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
}

func TestGenerateTokenPair(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	access, refresh, err := m.GenerateTokenPair("user-123")
	if err != nil {
		t.Fatalf("GenerateTokenPair returned error: %v", err)
	}
	if access == "" {
		t.Error("access token is empty")
	}
	if refresh == "" {
		t.Error("refresh token is empty")
	}

	// Validate the access token contains the correct userID
	userID, err := m.ValidateAccessToken(access)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error for generated token: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("expected userID %q, got %q", "user-123", userID)
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	token, err := m.generateAccessToken("user-456")
	if err != nil {
		t.Fatalf("generateAccessToken returned error: %v", err)
	}

	userID, err := m.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error: %v", err)
	}
	if userID != "user-456" {
		t.Errorf("expected userID %q, got %q", "user-456", userID)
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	// Manually create a token with expired claims
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user-789",
		"iat": now.Add(-2 * time.Hour).Unix(),
		"exp": now.Add(-1 * time.Hour).Unix(), // expired 1 hour ago
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = m.ValidateAccessToken(tokenStr)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	testCases := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"random string", "not-a-jwt-token"},
		{"partial jwt", "eyJhbGciOiJIUzI1NiJ9.bad.bad"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.ValidateAccessToken(tc.token)
			if err != ErrInvalidToken {
				t.Errorf("expected ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	rdb := setupMiniredis(t)
	m1 := NewJWTManager("secret-one", 15*time.Minute, 7*24*time.Hour, rdb)
	m2 := NewJWTManager("secret-two", 15*time.Minute, 7*24*time.Hour, rdb)

	token, err := m1.generateAccessToken("user-abc")
	if err != nil {
		t.Fatalf("generateAccessToken returned error: %v", err)
	}

	_, err = m2.ValidateAccessToken(token)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRotateRefreshToken(t *testing.T) {
	rdb := setupMiniredis(t)
	m := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, rdb)

	// Generate initial token pair
	_, refresh, err := m.GenerateTokenPair("user-rotate")
	if err != nil {
		t.Fatalf("GenerateTokenPair returned error: %v", err)
	}

	// Rotate using the refresh token
	ctx := context.Background()
	newAccess, newRefresh, err := m.RotateRefreshToken(ctx, refresh)
	if err != nil {
		t.Fatalf("RotateRefreshToken returned error: %v", err)
	}
	if newAccess == "" {
		t.Error("new access token is empty")
	}
	if newRefresh == "" {
		t.Error("new refresh token is empty")
	}

	// Validate new access token
	userID, err := m.ValidateAccessToken(newAccess)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error for rotated token: %v", err)
	}
	if userID != "user-rotate" {
		t.Errorf("expected userID %q, got %q", "user-rotate", userID)
	}

	// Old refresh token should be invalidated (already consumed)
	_, _, err = m.RotateRefreshToken(ctx, refresh)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for reused refresh token, got %v", err)
	}
}

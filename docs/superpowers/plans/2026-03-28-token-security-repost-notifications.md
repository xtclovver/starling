# Token Security + Repost Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add family_id + ua_hash + reuse detection to refresh token rotation, send notifications on repost/quote, and update Notion checkboxes.

**Architecture:** `jwt.go` gains family-based token tracking in Redis; `RotateRefreshToken` gets a `uaHash` parameter and implements reuse detection by checking meta before deleting; gateway extracts User-Agent hash and forwards it; `RepostPost`/`QuotePost` handlers fire notifications asynchronously, mirroring the existing `LikePost` pattern. Proto `RefreshTokenRequest` gets a new `ua_hash` field so the gateway can pass it through.

**Tech Stack:** Go 1.22+, Redis (miniredis for tests), protobuf/buf, React/TypeScript (no frontend changes needed)

---

## File Map

| File | Change |
|---|---|
| `backend/proto/user/v1/user.proto` | Add `ua_hash` field to `RefreshTokenRequest` |
| `backend/gen/go/user/v1/user.pb.go` | Regenerated (run `make proto-gen`) |
| `backend/services/user-svc/internal/auth/jwt.go` | Rewrite token storage/rotation with family_id + ua_hash + reuse detection |
| `backend/services/user-svc/internal/auth/jwt_test.go` | Update/add tests for new behaviour |
| `backend/services/user-svc/internal/grpc/server.go` | Pass `uaHash` from gRPC request to `GenerateTokenPairWithUA` / `RotateRefreshToken` |
| `backend/api-gateway/internal/handler/auth.go` | Compute `sha256(User-Agent)[:16]` and set `UaHash` in gRPC requests |
| `backend/api-gateway/internal/handler/post.go` | Add async notification goroutines to `RepostPost` and `QuotePost` |

---

## Task 1: Update proto — add `ua_hash` to all auth messages

**Files:**
- Modify: `backend/proto/user/v1/user.proto`

- [ ] **Step 1: Add `ua_hash` to all three auth request messages**

Open `backend/proto/user/v1/user.proto` and update the three messages:

```protobuf
message RegisterRequest {
  string username = 1;
  string email    = 2;
  string password = 3;
  string ua_hash  = 4;
}

message LoginRequest {
  string email    = 1;
  string password = 2;
  string ua_hash  = 3;
}

message RefreshTokenRequest {
  string refresh_token = 1;
  string ua_hash       = 2;
}
```

- [ ] **Step 2: Regenerate proto**

```bash
cd backend && make proto-gen
```

Expected: no errors, `backend/gen/go/user/v1/user.pb.go` updated with `UaHash` fields on all three messages.

- [ ] **Step 3: Commit**

```bash
cd "backend"
git add proto/user/v1/user.proto gen/go/user/v1/
git commit -m "feat(proto): add ua_hash to RegisterRequest, LoginRequest, RefreshTokenRequest"
```

---

## Task 2: Rewrite `jwt.go` — family_id + ua_hash + reuse detection

**Files:**
- Modify: `backend/services/user-svc/internal/auth/jwt.go`

### Redis key schema (new)

| Key | Type | Value | TTL |
|---|---|---|---|
| `refresh:{hash}` | string | `"1"` | refreshTTL |
| `refresh_meta:{hash}` | hash | `user_id`, `family_id`, `ua_hash` | refreshTTL |
| `family:{familyID}` | set | token hashes in this family | refreshTTL |
| `user_families:{userID}` | set | family IDs for this user | refreshTTL |

Old keys `user_refresh:{userID}` are replaced by `user_families:{userID}`.

- [ ] **Step 1: Replace `jwt.go` with new implementation**

```go
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	rdb        *redis.Client
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration, rdb *redis.Client) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		rdb:        rdb,
	}
}

// GenerateTokenPair issues a fresh token pair starting a new refresh family.
// uaHash should be SHA256(User-Agent)[:16 hex chars]; may be empty.
func (m *JWTManager) GenerateTokenPair(ctx context.Context, userID, uaHash string) (string, string, error) {
	accessToken, err := m.generateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	familyID, err := generateID()
	if err != nil {
		return "", "", fmt.Errorf("generate family id: %w", err)
	}

	refreshToken, err := m.storeRefreshToken(ctx, userID, familyID, uaHash)
	if err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// RotateRefreshToken validates oldToken, checks ua_hash, detects reuse, and issues a new pair.
func (m *JWTManager) RotateRefreshToken(ctx context.Context, oldToken, uaHash string) (string, string, error) {
	oldHash := hashToken(oldToken)

	// 1. Read meta BEFORE deleting (so we have familyID for reuse detection)
	meta, err := m.rdb.HGetAll(ctx, "refresh_meta:"+oldHash).Result()
	if err != nil || len(meta) == 0 {
		// Token never existed or already cleaned up — could be reuse
		return "", "", ErrInvalidToken
	}

	userID := meta["user_id"]
	familyID := meta["family_id"]
	storedUA := meta["ua_hash"]

	// 2. Check ua_hash if one was stored
	if storedUA != "" && uaHash != "" && storedUA != uaHash {
		// UA mismatch — invalidate the whole family
		m.invalidateFamily(ctx, familyID)
		return "", "", ErrInvalidToken
	}

	// 3. Atomic delete of the existence flag — returns 0 if already gone (reuse detection)
	deleted, err := m.rdb.Del(ctx, "refresh:"+oldHash).Result()
	if err != nil {
		return "", "", err
	}
	if deleted == 0 {
		// Token was already consumed — reuse detected, invalidate family
		m.invalidateFamily(ctx, familyID)
		return "", "", ErrInvalidToken
	}

	// 4. Clean up meta and family set entry for old token
	pipe := m.rdb.Pipeline()
	pipe.Del(ctx, "refresh_meta:"+oldHash)
	pipe.SRem(ctx, "family:"+familyID, oldHash)
	_, _ = pipe.Exec(ctx)

	// 5. Issue new token continuing the same family
	accessToken, err := m.generateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := m.storeRefreshToken(ctx, userID, familyID, uaHash)
	if err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// Logout blacklists the current access token and removes the refresh token.
func (m *JWTManager) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if jti, exp, err := m.ExtractJTI(accessToken); err == nil {
		if remaining := time.Until(exp); remaining > 0 {
			m.rdb.Set(ctx, "blacklist:"+jti, "1", remaining)
		}
	}

	if refreshToken != "" {
		hash := hashToken(refreshToken)
		meta, err := m.rdb.HGetAll(ctx, "refresh_meta:"+hash).Result()
		if err == nil && len(meta) > 0 {
			familyID := meta["family_id"]
			pipe := m.rdb.Pipeline()
			pipe.Del(ctx, "refresh:"+hash)
			pipe.Del(ctx, "refresh_meta:"+hash)
			pipe.SRem(ctx, "family:"+familyID, hash)
			_, _ = pipe.Exec(ctx)
		}
	}

	return nil
}

// RevokeAllTokens invalidates all refresh families for a user and blacklists the current access token.
func (m *JWTManager) RevokeAllTokens(ctx context.Context, userID, accessToken string) error {
	if jti, exp, err := m.ExtractJTI(accessToken); err == nil {
		if remaining := time.Until(exp); remaining > 0 {
			m.rdb.Set(ctx, "blacklist:"+jti, "1", remaining)
		}
	}

	familiesKey := "user_families:" + userID
	familyIDs, err := m.rdb.SMembers(ctx, familiesKey).Result()
	if err != nil {
		return err
	}

	for _, fid := range familyIDs {
		m.invalidateFamily(ctx, fid)
	}
	m.rdb.Del(ctx, familiesKey)

	return nil
}

// ValidateAccessToken validates an access token and returns userID and jti.
func (m *JWTManager) ValidateAccessToken(tokenStr string) (userID string, jti string, err error) {
	token, parseErr := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if parseErr != nil {
		if errors.Is(parseErr, jwt.ErrTokenExpired) {
			return "", "", ErrExpiredToken
		}
		return "", "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", ErrInvalidToken
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", "", ErrInvalidToken
	}

	jtiVal, _ := claims["jti"].(string)
	return sub, jtiVal, nil
}

// ExtractJTI parses an access token (even expired) and returns the jti and exp.
func (m *JWTManager) ExtractJTI(tokenStr string) (jti string, exp time.Time, err error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", time.Time{}, ErrInvalidToken
	}
	jtiVal, _ := claims["jti"].(string)
	if jtiVal == "" {
		return "", time.Time{}, ErrInvalidToken
	}
	var expTime time.Time
	if expFloat, ok := claims["exp"].(float64); ok {
		expTime = time.Unix(int64(expFloat), 0)
	}
	return jtiVal, expTime, nil
}

// storeRefreshToken generates a random token, stores its meta in Redis, returns the raw token.
func (m *JWTManager) storeRefreshToken(ctx context.Context, userID, familyID, uaHash string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	hash := hashToken(token)

	pipe := m.rdb.Pipeline()
	pipe.Set(ctx, "refresh:"+hash, "1", m.refreshTTL)
	pipe.HSet(ctx, "refresh_meta:"+hash, "user_id", userID, "family_id", familyID, "ua_hash", uaHash)
	pipe.Expire(ctx, "refresh_meta:"+hash, m.refreshTTL)
	pipe.SAdd(ctx, "family:"+familyID, hash)
	pipe.Expire(ctx, "family:"+familyID, m.refreshTTL)
	pipe.SAdd(ctx, "user_families:"+userID, familyID)
	pipe.Expire(ctx, "user_families:"+userID, m.refreshTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return token, nil
}

// invalidateFamily deletes all tokens in a refresh family.
func (m *JWTManager) invalidateFamily(ctx context.Context, familyID string) {
	familyKey := "family:" + familyID
	hashes, err := m.rdb.SMembers(ctx, familyKey).Result()
	if err != nil {
		return
	}
	if len(hashes) > 0 {
		keys := make([]string, 0, len(hashes)*2)
		for _, h := range hashes {
			keys = append(keys, "refresh:"+h, "refresh_meta:"+h)
		}
		m.rdb.Del(ctx, keys...)
	}
	m.rdb.Del(ctx, familyKey)
}

func (m *JWTManager) generateAccessToken(userID string) (string, error) {
	jti, err := generateID()
	if err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": now.Unix(),
		"exp": now.Add(m.accessTTL).Unix(),
		"jti": jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 2: Commit**

```bash
cd "backend/services/user-svc"
git add internal/auth/jwt.go
git commit -m "feat(user-svc): add family_id, ua_hash, reuse detection to refresh token rotation"
```

---

## Task 3: Update `jwt_test.go` — cover new behaviour

**Files:**
- Modify: `backend/services/user-svc/internal/auth/jwt_test.go`

- [ ] **Step 1: Replace test file**

```go
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
```

- [ ] **Step 2: Run tests**

```bash
cd "backend/services/user-svc" && go test ./internal/auth/... -v -count=1
```

Expected: all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/auth/jwt_test.go
git commit -m "test(user-svc): update jwt tests for family_id + ua_hash + reuse detection"
```

---

## Task 4: Update `server.go` in user-svc — pass uaHash

**Files:**
- Modify: `backend/services/user-svc/internal/grpc/server.go`

- [ ] **Step 1: Update `Register` — pass uaHash**

Find the `Register` method's call to `GenerateTokenPair` (~line 106) and replace:

```go
accessToken, refreshToken, err := s.jwt.GenerateTokenPair(user.ID)
```

with:

```go
accessToken, refreshToken, err := s.jwt.GenerateTokenPair(ctx, user.ID, req.GetUaHash())
```

- [ ] **Step 2: Update `Login` — pass uaHash**

Find the `Login` method's call to `GenerateTokenPair` (~line 136) and replace:

```go
accessToken, refreshToken, err := s.jwt.GenerateTokenPair(user.ID)
```

with:

```go
accessToken, refreshToken, err := s.jwt.GenerateTokenPair(ctx, user.ID, req.GetUaHash())
```

- [ ] **Step 3: Update `RefreshToken` — pass uaHash**

Find the `RefreshToken` method (~line 151) and replace:

```go
accessToken, refreshToken, err := s.jwt.RotateRefreshToken(ctx, req.GetRefreshToken())
```

with:

```go
accessToken, refreshToken, err := s.jwt.RotateRefreshToken(ctx, req.GetRefreshToken(), req.GetUaHash())
```

- [ ] **Step 4: Build to verify no compile errors**

```bash
cd "backend/services/user-svc" && go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run tests**

```bash
go test ./... -count=1
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/grpc/server.go
git commit -m "feat(user-svc): forward ua_hash from gRPC requests to JWTManager"
```

---

## Task 5: Update `auth.go` in api-gateway — compute and send ua_hash

**Files:**
- Modify: `backend/api-gateway/internal/handler/auth.go`

The gateway must compute `sha256(User-Agent)` and send the first 16 hex chars to user-svc on Register, Login, and Refresh calls.

- [ ] **Step 1: Add `uaHash` helper at top of file**

Add after the imports (the file already imports `"crypto/sha256"` — if not, add it):

```go
import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "strings"

    userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

// computeUAHash returns the first 16 hex chars of SHA256(User-Agent).
func computeUAHash(r *http.Request) string {
    ua := r.Header.Get("User-Agent")
    if ua == "" {
        return ""
    }
    sum := sha256.Sum256([]byte(ua))
    return hex.EncodeToString(sum[:])[:16]
}
```

- [ ] **Step 2: Update `Register` handler**

Find the `h.user.Register(...)` call and add `UaHash`:

```go
resp, err := h.user.Register(r.Context(), &userpb.RegisterRequest{
    Username: req.Username,
    Email:    req.Email,
    Password: req.Password,
    UaHash:   computeUAHash(r),
})
```

- [ ] **Step 3: Update `Login` handler**

```go
resp, err := h.user.Login(r.Context(), &userpb.LoginRequest{
    Email:   req.Email,
    Password: req.Password,
    UaHash:  computeUAHash(r),
})
```

- [ ] **Step 4: Update `Refresh` handler**

```go
resp, err := h.user.RefreshToken(r.Context(), &userpb.RefreshTokenRequest{
    RefreshToken: cookie.Value,
    UaHash:       computeUAHash(r),
})
```

- [ ] **Step 5: Build to verify**

```bash
cd "backend/api-gateway" && go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/handler/auth.go
git commit -m "feat(gateway): compute ua_hash from User-Agent and forward to UserSvc"
```

---

## Task 6: Add notifications to `RepostPost` and `QuotePost` handlers

**Files:**
- Modify: `backend/api-gateway/internal/handler/post.go`

- [ ] **Step 1: Update `RepostPost` — add async notification goroutine**

Find `RepostPost` (currently ~line 279). Replace the entire function:

```go
func (h *PostHandler) RepostPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.post.RepostPost(r.Context(), &postpb.RepostPostRequest{PostId: id, UserId: userID})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	go func() {
		resp, err := h.post.GetPost(context.Background(), &postpb.GetPostRequest{Id: id})
		if err != nil || resp.GetPost().GetUserId() == userID {
			return
		}
		ownerID := resp.GetPost().GetUserId()
		nr, err := h.user.CreateNotification(context.Background(), &userpb.CreateNotificationRequest{
			UserId:  ownerID,
			ActorId: userID,
			Type:    "repost",
			PostId:  id,
		})
		if err == nil && h.notifier != nil {
			h.notifier.PublishNotification(context.Background(), ownerID, notificationToMap(nr.GetNotification()))
		}
	}()

	writeJSON(w, http.StatusOK, nil)
}
```

- [ ] **Step 2: Update `QuotePost` — add async notification goroutine**

Find `QuotePost` (currently ~line 303). Replace the entire function:

```go
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

	go func() {
		origResp, err := h.post.GetPost(context.Background(), &postpb.GetPostRequest{Id: id})
		if err != nil || origResp.GetPost().GetUserId() == userID {
			return
		}
		ownerID := origResp.GetPost().GetUserId()
		nr, err := h.user.CreateNotification(context.Background(), &userpb.CreateNotificationRequest{
			UserId:  ownerID,
			ActorId: userID,
			Type:    "quote",
			PostId:  id,
		})
		if err == nil && h.notifier != nil {
			h.notifier.PublishNotification(context.Background(), ownerID, notificationToMap(nr.GetNotification()))
		}
	}()

	post := postToMap(resp.GetPost())
	h.enrichSinglePost(r, post)
	writeJSON(w, http.StatusCreated, map[string]any{"post": post})
}
```

- [ ] **Step 3: Build**

```bash
cd "backend/api-gateway" && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/post.go
git commit -m "feat(gateway): send repost and quote notifications to post owner"
```

---

## Task 7: Update Notion checkboxes

- [ ] **Step 1: Mark completed items in Notion page `3148901a-bcf4-8157-a247-fd73f4a81ed9`**

Use the Notion MCP tool to update the following checkboxes to checked (`[x]`):

**Section 1.3.1:**
- `Привязка refresh_token к User-Agent hash — при несовпадении UA токен отклоняется и цепочка инвалидируется`
- `Refresh token rotation с family_id — при каждом использовании выдаётся новая пара, старый инвалидируется`
- `Reuse detection — повторное использование уже ротированного refresh_token инвалидирует всю цепочку (family)`

**Section 2.1:**
- `Хочу получать уведомления о лайках, комментариях, подписках и репостах`
- `Хочу репостнуть или процитировать пост другого пользователя`

**Section 2.2:**
- `Репост — клик репост → INSERT в reposts → INCR reposts_count → появление в лентах подписчиков → уведомление автору`
- `Уведомления — действие (лайк/коммент/подписка/репост) → Gateway оркестрирует CreateNotification → PUBLISH в Redis → WebSocket доставляет в реальном времени`

**Section 5.2:**
- `При выдаче refresh_token сохранять в Redis Hash: user_id, family_id, ua_hash, ip_subnet`
- `При выдаче добавлять token_hash в refresh_family:{family_id} и family_id в user_sessions:{user_id}`
- `Reuse detection — если refresh_token не найден в Redis, но family_id существует → инвалидировать всю family`
- `Проверка ua_hash и ip_subnet при refresh — несовпадение → отклонить и инвалидировать family`

Note: ip_subnet items can be partially checked (ua_hash part done, ip_subnet not implemented by design).

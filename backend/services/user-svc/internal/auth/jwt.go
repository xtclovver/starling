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

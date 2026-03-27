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

func (m *JWTManager) GenerateTokenPair(userID string) (string, string, error) {
	accessToken, err := m.generateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := m.generateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

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

func (m *JWTManager) RotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {
	oldHash := hashToken(oldToken)
	key := "refresh:" + oldHash

	userID, err := m.rdb.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", "", ErrInvalidToken
		}
		return "", "", err
	}

	// Remove old hash from user's refresh set
	m.rdb.SRem(ctx, "user_refresh:"+userID, oldHash)

	return m.GenerateTokenPair(userID)
}

func (m *JWTManager) generateAccessToken(userID string) (string, error) {
	jti, err := generateJTI()
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

func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *JWTManager) generateRefreshToken(userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	hash := hashToken(token)
	key := "refresh:" + hash
	setKey := "user_refresh:" + userID

	ctx := context.Background()
	pipe := m.rdb.Pipeline()
	pipe.Set(ctx, key, userID, m.refreshTTL)
	pipe.SAdd(ctx, setKey, hash)
	pipe.Expire(ctx, setKey, m.refreshTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return token, nil
}

// Logout blacklists the current access token and deletes the refresh token.
func (m *JWTManager) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// Blacklist the access token's jti
	if jti, exp, err := m.ExtractJTI(accessToken); err == nil {
		remaining := time.Until(exp)
		if remaining > 0 {
			m.rdb.Set(ctx, "blacklist:"+jti, "1", remaining)
		}
	}

	// Delete the refresh token
	if refreshToken != "" {
		hash := hashToken(refreshToken)
		// Get userID before deleting so we can clean the set
		userID, err := m.rdb.GetDel(ctx, "refresh:"+hash).Result()
		if err == nil && userID != "" {
			m.rdb.SRem(ctx, "user_refresh:"+userID, hash)
		}
	}

	return nil
}

// RevokeAllTokens deletes all refresh tokens for a user and blacklists the current access token.
func (m *JWTManager) RevokeAllTokens(ctx context.Context, userID, accessToken string) error {
	// Blacklist the current access token's jti
	if jti, exp, err := m.ExtractJTI(accessToken); err == nil {
		remaining := time.Until(exp)
		if remaining > 0 {
			m.rdb.Set(ctx, "blacklist:"+jti, "1", remaining)
		}
	}

	// Delete all refresh tokens for this user
	setKey := "user_refresh:" + userID
	hashes, err := m.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return err
	}

	if len(hashes) > 0 {
		keys := make([]string, len(hashes))
		for i, h := range hashes {
			keys[i] = "refresh:" + h
		}
		m.rdb.Del(ctx, keys...)
	}
	m.rdb.Del(ctx, setKey)

	return nil
}

// ExtractJTI parses an access token (even expired) and returns the jti and exp.
// Signature is verified; expiration is intentionally skipped so we can blacklist
// tokens that are still valid for other holders.
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

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

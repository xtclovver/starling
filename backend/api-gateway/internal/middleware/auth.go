package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const UserIDKey ctxKey = "user_id"

type Auth struct {
	secret []byte
	rdb    *redis.Client
}

func NewAuth(secret string, rdb *redis.Client) *Auth {
	return &Auth{secret: []byte(secret), rdb: rdb}
}

func (a *Auth) Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := a.extractUserID(r)
		if err != nil {
			http.Error(w, `{"data":null,"error":{"code":401,"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID, err := a.extractUserID(r); err == nil {
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Auth) extractUserID(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", jwt.ErrTokenMalformed
	}
	tokenStr := strings.TrimPrefix(header, "Bearer ")

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return a.secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	// Check jti blacklist
	if jti, ok := claims["jti"].(string); ok && jti != "" && a.rdb != nil {
		if exists, _ := a.rdb.Exists(r.Context(), "blacklist:"+jti).Result(); exists > 0 {
			return "", jwt.ErrTokenInvalidClaims
		}
	}

	return sub, nil
}

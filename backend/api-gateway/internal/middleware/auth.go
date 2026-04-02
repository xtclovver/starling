package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)

const UserIDKey ctxKey = "user_id"
const IsAdminKey ctxKey = "is_admin"

type Auth struct {
	secret []byte
	rdb    *redis.Client
	user   userpb.UserServiceClient
}

func NewAuth(secret string, rdb *redis.Client, user userpb.UserServiceClient) *Auth {
	return &Auth{secret: []byte(secret), rdb: rdb, user: user}
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

func (a *Auth) AdminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := a.extractUserID(r)
		if err != nil {
			http.Error(w, `{"data":null,"error":{"code":401,"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}

		// Check is_admin from database via gRPC, not from JWT claims
		resp, err := a.user.GetUser(r.Context(), &userpb.GetUserRequest{Id: userID})
		if err != nil || !resp.GetUser().GetIsAdmin() {
			http.Error(w, `{"data":null,"error":{"code":403,"message":"admin access required"}}`, http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, IsAdminKey, true)
		next.ServeHTTP(w, r.WithContext(ctx))
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

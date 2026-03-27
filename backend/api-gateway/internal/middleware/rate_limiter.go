package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

func (rl *RateLimiter) Limit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			key := fmt.Sprintf("rate_limit:%s:%s", ip, r.URL.Path)

			ctx := r.Context()
			count, err := rl.rdb.Incr(ctx, key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rl.rdb.Expire(ctx, key, window)
			}

			if count > int64(maxRequests) {
				ttl, _ := rl.rdb.TTL(ctx, key).Result()
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())+1))
				http.Error(w, `{"data":null,"error":{"code":429,"message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) AuthLimit() func(http.Handler) http.Handler {
	return rl.Limit(5, time.Minute)
}

func (rl *RateLimiter) UploadLimit() func(http.Handler) http.Handler {
	return rl.Limit(10, time.Minute)
}

func (rl *RateLimiter) DefaultAuth() func(http.Handler) http.Handler {
	return rl.Limit(100, time.Minute)
}

func (rl *RateLimiter) Guest() func(http.Handler) http.Handler {
	return rl.Limit(60, time.Minute)
}

// Adaptive applies a higher rate limit to authenticated requests and a lower one to guests.
func (rl *RateLimiter) Adaptive(guestMax, authMax int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			maxReqs := guestMax
			prefix := "rl:guest:"
			if r.Header.Get("Authorization") != "" {
				maxReqs = authMax
				prefix = "rl:auth:"
			}
			key := fmt.Sprintf("%s%s", prefix, ip)

			ctx := r.Context()
			count, err := rl.rdb.Incr(ctx, key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				rl.rdb.Expire(ctx, key, window)
			}
			if count > int64(maxReqs) {
				ttl, _ := rl.rdb.TTL(ctx, key).Result()
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())+1))
				http.Error(w, `{"data":null,"error":{"code":429,"message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

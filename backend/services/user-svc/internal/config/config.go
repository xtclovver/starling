package config

import (
	"os"
	"time"
)

type Config struct {
	DBUrl          string
	RedisURL       string
	GRPCPort       string
	JWTSecret      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration
	PostSvcAddr    string
}

func Load() *Config {
	return &Config{
		DBUrl:         getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/microtwitter?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		JWTAccessTTL:  parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
		JWTRefreshTTL: parseDuration(getEnv("JWT_REFRESH_TTL", "168h")),
		PostSvcAddr:   getEnv("POST_SVC_ADDR", "localhost:50052"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

package config

import (
	"os"
	"time"
)

type Config struct {
	DBUrl            string
	RedisURL         string
	GRPCPort         string
	LikeSyncInterval time.Duration
	ViewSyncInterval time.Duration
}

func Load() *Config {
	return &Config{
		DBUrl:            getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/microtwitter?sslmode=disable"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		GRPCPort:         getEnv("GRPC_PORT", "50052"),
		LikeSyncInterval: parseDuration(getEnv("LIKE_SYNC_INTERVAL", "30s")),
		ViewSyncInterval: parseDuration(getEnv("VIEW_SYNC_INTERVAL", "60s")),
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
		return 30 * time.Second
	}
	return d
}

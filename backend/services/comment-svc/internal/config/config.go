package config

import "os"

type Config struct {
	DBUrl    string
	GRPCPort string
}

func Load() *Config {
	return &Config{
		DBUrl:    getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/microtwitter?sslmode=disable"),
		GRPCPort: getEnv("GRPC_PORT", "50053"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

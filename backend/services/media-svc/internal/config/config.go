package config

import "os"

type Config struct {
	DBUrl               string
	GRPCPort            string
	MinIOEndpoint       string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOBucket         string
	MinIOUseSSL         bool
}

func Load() *Config {
	return &Config{
		DBUrl:          getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/microtwitter?sslmode=disable"),
		GRPCPort:       getEnv("GRPC_PORT", "50054"),
		MinIOEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "media"),
		MinIOUseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

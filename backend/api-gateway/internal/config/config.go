package config

import "os"

type Config struct {
	HTTPPort       string
	UserSvcAddr    string
	PostSvcAddr    string
	CommentSvcAddr string
	MediaSvcAddr   string
	RedisURL       string
	JWTSecret      string
	CORSOrigin     string
	CookieSecure   bool
}

func Load() *Config {
	return &Config{
		HTTPPort:       getEnv("PORT", "8080"),
		UserSvcAddr:    getEnv("USER_SVC_ADDR", "localhost:50051"),
		PostSvcAddr:    getEnv("POST_SVC_ADDR", "localhost:50052"),
		CommentSvcAddr: getEnv("COMMENT_SVC_ADDR", "localhost:50053"),
		MediaSvcAddr:   getEnv("MEDIA_SVC_ADDR", "localhost:50054"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
		CORSOrigin:     getEnv("CORS_ORIGIN", "http://localhost:3000"),
		CookieSecure:   getEnv("COOKIE_SECURE", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

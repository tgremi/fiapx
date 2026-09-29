package config

import (
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
}

func Load() Config {
	return Config{
		Port:        getEnv("AUTH_PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://fiapx:fiapx@localhost:5432/fiapx?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiry:   24 * time.Hour,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

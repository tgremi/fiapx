package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AUTH_PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	c := Load()
	if c.Port != "8081" {
		t.Fatalf("port = %q", c.Port)
	}
	if c.DatabaseURL == "" {
		t.Fatal("database url vazia")
	}
	if c.JWTExpiry != 24*time.Hour {
		t.Fatalf("expiry = %v", c.JWTExpiry)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("AUTH_PORT", "9999")
	t.Setenv("JWT_SECRET", "meu-segredo")

	c := Load()
	if c.Port != "9999" {
		t.Fatalf("port = %q", c.Port)
	}
	if c.JWTSecret != "meu-segredo" {
		t.Fatalf("secret = %q", c.JWTSecret)
	}
}

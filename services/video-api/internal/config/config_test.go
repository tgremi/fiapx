package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VIDEO_API_PORT", "")
	t.Setenv("STORAGE_DIR", "")
	t.Setenv("REDIS_ADDR", "")

	c := Load()
	if c.Port != "8080" {
		t.Fatalf("port = %q", c.Port)
	}
	if c.StorageDir != "./data" {
		t.Fatalf("storage dir = %q", c.StorageDir)
	}
	if c.RedisAddr != "localhost:6379" {
		t.Fatalf("redis = %q", c.RedisAddr)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("VIDEO_API_PORT", "9999")
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@broker:5672/")

	c := Load()
	if c.Port != "9999" {
		t.Fatalf("port = %q", c.Port)
	}
	if c.RabbitMQURL != "amqp://guest:guest@broker:5672/" {
		t.Fatalf("rabbit = %q", c.RabbitMQURL)
	}
}

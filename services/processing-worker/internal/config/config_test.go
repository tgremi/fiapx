package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("STORAGE_DIR", "")
	t.Setenv("RABBITMQ_URL", "")

	c := Load()
	if c.StorageDir != "./data" {
		t.Fatalf("storage dir = %q", c.StorageDir)
	}
	if c.RabbitMQURL != "amqp://fiapx:fiapx@localhost:5672/" {
		t.Fatalf("rabbit = %q", c.RabbitMQURL)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@host/db")
	t.Setenv("STORAGE_DIR", "/tmp/frames")

	c := Load()
	if c.DatabaseURL != "postgres://u:p@host/db" {
		t.Fatalf("database = %q", c.DatabaseURL)
	}
	if c.StorageDir != "/tmp/frames" {
		t.Fatalf("storage dir = %q", c.StorageDir)
	}
}

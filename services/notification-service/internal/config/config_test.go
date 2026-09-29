package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_FROM", "")

	c := Load()
	if c.SMTPHost != "localhost" {
		t.Fatalf("smtp host = %q", c.SMTPHost)
	}
	if c.SMTPPort != "1025" {
		t.Fatalf("smtp port = %q", c.SMTPPort)
	}
	if c.SMTPFrom != "no-reply@fiapx.com" {
		t.Fatalf("smtp from = %q", c.SMTPFrom)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SMTP_HOST", "mailhog")
	t.Setenv("RABBITMQ_URL", "amqp://broker")

	c := Load()
	if c.SMTPHost != "mailhog" {
		t.Fatalf("smtp host = %q", c.SMTPHost)
	}
	if c.RabbitMQURL != "amqp://broker" {
		t.Fatalf("rabbit = %q", c.RabbitMQURL)
	}
}

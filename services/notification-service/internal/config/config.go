package config

import "os"

type Config struct {
	DatabaseURL string
	RabbitMQURL string
	SMTPHost    string
	SMTPPort    string
	SMTPFrom    string
	MetricsPort string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://fiapx:fiapx@localhost:5432/fiapx?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://fiapx:fiapx@localhost:5672/"),
		SMTPHost:    getEnv("SMTP_HOST", "localhost"),
		SMTPPort:    getEnv("SMTP_PORT", "1025"),
		SMTPFrom:    getEnv("SMTP_FROM", "no-reply@fiapx.com"),
		MetricsPort: getEnv("METRICS_PORT", "9091"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

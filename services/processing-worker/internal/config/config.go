package config

import "os"

type Config struct {
	DatabaseURL   string
	StorageDir    string
	StorageDriver string
	S3Endpoint    string
	S3AccessKey   string
	S3SecretKey   string
	S3Bucket      string
	S3Region      string
	RabbitMQURL   string
	MetricsPort   string
}

func Load() Config {
	return Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://fiapx:fiapx@localhost:5432/fiapx?sslmode=disable"),
		StorageDir:    getEnv("STORAGE_DIR", "./data"),
		StorageDriver: getEnv("STORAGE_DRIVER", "s3"),
		S3Endpoint:    getEnv("S3_ENDPOINT", "localhost:8333"),
		S3AccessKey:   getEnv("S3_ACCESS_KEY", "test"),
		S3SecretKey:   getEnv("S3_SECRET_KEY", "test"),
		S3Bucket:      getEnv("S3_BUCKET", "fiapx"),
		S3Region:      getEnv("S3_REGION", "us-east-1"),
		RabbitMQURL:   getEnv("RABBITMQ_URL", "amqp://fiapx:fiapx@localhost:5672/"),
		MetricsPort:   getEnv("METRICS_PORT", "9090"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

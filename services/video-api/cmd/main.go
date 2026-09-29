package main

import (
	"context"
	"log"
	"time"

	"github.com/fiapx/video-api/internal/adapters/in/rest"
	"github.com/fiapx/video-api/internal/adapters/out/postgres"
	"github.com/fiapx/video-api/internal/adapters/out/rabbitmq"
	"github.com/fiapx/video-api/internal/adapters/out/redis"
	"github.com/fiapx/video-api/internal/adapters/out/storage"
	"github.com/fiapx/video-api/internal/adapters/out/token"
	"github.com/fiapx/video-api/internal/application"
	"github.com/fiapx/video-api/internal/config"
	"github.com/fiapx/video-api/internal/db"
	"github.com/fiapx/video-api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("falha ao conectar no banco: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(cfg.DatabaseURL, "video_schema_migrations"); err != nil {
		log.Fatalf("falha ao rodar migrations: %v", err)
	}

	publisher, err := rabbitmq.NewPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("falha ao conectar no RabbitMQ: %v", err)
	}
	defer publisher.Close()

	cache := redis.NewCache(cfg.RedisAddr, 30*time.Second)
	defer func() { _ = cache.Close() }()

	videoRepo := postgres.NewVideoRepository(pool)
	objectStorage := newObjectStorage(cfg)
	validator := token.NewValidator(cfg.JWTSecret)

	usecase := application.NewVideoUseCase(videoRepo, objectStorage, cache, publisher)
	handler := rest.NewHandler(usecase, validator)

	log.Printf("video-api ouvindo na porta %s", cfg.Port)
	if err := handler.Router().Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

func newObjectStorage(cfg config.Config) domain.ObjectStorage {
	if cfg.StorageDriver == "filesystem" {
		return storage.NewFilesystemStorage(cfg.StorageDir)
	}

	s3, err := storage.NewS3Storage(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Region)
	if err != nil {
		log.Fatalf("falha ao conectar no S3: %v", err)
	}
	return s3
}

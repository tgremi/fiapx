package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/fiapx/processing-worker/internal/adapters/in/amqp"
	"github.com/fiapx/processing-worker/internal/adapters/out/ffmpeg"
	"github.com/fiapx/processing-worker/internal/adapters/out/postgres"
	"github.com/fiapx/processing-worker/internal/adapters/out/rabbitmq"
	"github.com/fiapx/processing-worker/internal/adapters/out/storage"
	"github.com/fiapx/processing-worker/internal/application"
	"github.com/fiapx/processing-worker/internal/config"
	"github.com/fiapx/processing-worker/internal/domain"
	"github.com/fiapx/processing-worker/internal/metrics"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("falha ao conectar no banco: %v", err)
	}
	defer pool.Close()

	publisher, err := rabbitmq.NewPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("falha ao conectar no RabbitMQ: %v", err)
	}
	defer publisher.Close()

	repo := postgres.NewVideoRepository(pool)
	objectStorage := newObjectStorage(cfg)
	processor := ffmpeg.NewProcessor()

	usecase := application.NewProcessingUseCase(processor, objectStorage, repo, publisher)

	consumer, err := amqp.NewConsumer(cfg.RabbitMQURL, usecase)
	if err != nil {
		log.Fatalf("falha ao iniciar consumidor: %v", err)
	}
	defer consumer.Close()

	go func() {
		log.Printf("processing-worker expondo métricas em :%s", cfg.MetricsPort)
		if err := metrics.Serve(":" + cfg.MetricsPort); err != nil {
			log.Printf("servidor de métricas encerrou: %v", err)
		}
	}()

	log.Println("processing-worker consumindo a fila video.processing")
	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("consumidor encerrou com erro: %v", err)
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

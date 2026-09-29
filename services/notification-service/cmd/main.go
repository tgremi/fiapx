package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/fiapx/notification-service/internal/adapters/in/amqp"
	"github.com/fiapx/notification-service/internal/adapters/out/postgres"
	"github.com/fiapx/notification-service/internal/adapters/out/smtp"
	"github.com/fiapx/notification-service/internal/application"
	"github.com/fiapx/notification-service/internal/config"
	"github.com/fiapx/notification-service/internal/metrics"
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

	userRepo := postgres.NewUserRepository(pool)
	notifier := smtp.NewSMTPNotifier(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom)

	usecase := application.NewNotificationUseCase(notifier, userRepo)

	consumer, err := amqp.NewConsumer(cfg.RabbitMQURL, usecase)
	if err != nil {
		log.Fatalf("falha ao iniciar consumidor: %v", err)
	}
	defer consumer.Close()

	go func() {
		log.Printf("notification-service expondo métricas em :%s", cfg.MetricsPort)
		if err := metrics.Serve(":" + cfg.MetricsPort); err != nil {
			log.Printf("servidor de métricas encerrou: %v", err)
		}
	}()

	log.Println("notification-service consumindo a fila notification")
	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("consumidor encerrou com erro: %v", err)
	}
}

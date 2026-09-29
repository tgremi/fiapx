package main

import (
	"context"
	"log"

	"github.com/fiapx/auth-service/internal/adapters/in/rest"
	"github.com/fiapx/auth-service/internal/adapters/out/hasher"
	"github.com/fiapx/auth-service/internal/adapters/out/postgres"
	"github.com/fiapx/auth-service/internal/adapters/out/token"
	"github.com/fiapx/auth-service/internal/application"
	"github.com/fiapx/auth-service/internal/config"
	"github.com/fiapx/auth-service/internal/db"
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

	if err := db.RunMigrations(cfg.DatabaseURL, "auth_schema_migrations"); err != nil {
		log.Fatalf("falha ao rodar migrations: %v", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	passwordHasher := hasher.NewPasswordHasher()
	tokenService := token.NewTokenService(cfg.JWTSecret, cfg.JWTExpiry)

	auth := application.NewAuthUseCase(userRepo, passwordHasher, tokenService)

	handler := rest.NewHandler(auth)

	log.Printf("auth-service ouvindo na porta %s", cfg.Port)
	if err := handler.Router().Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

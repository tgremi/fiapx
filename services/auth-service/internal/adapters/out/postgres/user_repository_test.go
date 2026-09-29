//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/fiapx/auth-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pg, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("fiapx"),
		postgres.WithUsername("fiapx"),
		postgres.WithPassword("fiapx"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	for _, stmt := range []string{
		`CREATE SCHEMA IF NOT EXISTS auth`,
		`CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	return pool
}

func TestUserRepositoryCreateAndFind(t *testing.T) {
	repo := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	u, err := repo.Create(ctx, "a@b.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == "" || u.Email != "a@b.com" || u.PasswordHash != "hash" {
		t.Fatalf("dados inesperados: %+v", u)
	}

	if _, err := repo.Create(ctx, "a@b.com", "hash2"); err != domain.ErrUserAlreadyExists {
		t.Fatalf("esperado ErrUserAlreadyExists, got %v", err)
	}

	found, err := repo.FindByEmail(ctx, "a@b.com")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.ID != u.ID {
		t.Fatalf("id diverge: %s != %s", found.ID, u.ID)
	}

	if _, err := repo.FindByEmail(ctx, "nao@existe.com"); err != domain.ErrUserNotFound {
		t.Fatalf("esperado ErrUserNotFound, got %v", err)
	}
}

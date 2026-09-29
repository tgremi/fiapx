//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/fiapx/notification-service/internal/domain"
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
			id UUID PRIMARY KEY,
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

func TestUserRepositoryFindByID(t *testing.T) {
	pool := newTestDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	const id = "11111111-1111-1111-1111-111111111111"
	_, err := pool.Exec(ctx,
		`INSERT INTO auth.users (id, email, password_hash) VALUES ($1::uuid, 'u@fiapx.com', 'hash')`,
		id,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	u, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if u.ID != id || u.Email != "u@fiapx.com" {
		t.Fatalf("dados inesperados: %+v", u)
	}

	if _, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); err != domain.ErrUserNotFound {
		t.Fatalf("esperado ErrUserNotFound, got %v", err)
	}
}

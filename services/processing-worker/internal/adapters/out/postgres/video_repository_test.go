//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/fiapx/processing-worker/internal/domain"
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
		`CREATE SCHEMA IF NOT EXISTS video`,
		`CREATE TYPE video.video_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')`,
		`CREATE TABLE IF NOT EXISTS video.videos (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			status video.video_status NOT NULL DEFAULT 'PENDING',
			original_key VARCHAR(512) NOT NULL,
			zip_key VARCHAR(512),
			error_message TEXT,
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

func TestVideoRepositoryUpdateAndGetStatus(t *testing.T) {
	pool := newTestDB(t)
	repo := NewVideoRepository(pool)
	ctx := context.Background()

	const id = "11111111-1111-1111-1111-111111111111"
	_, err := pool.Exec(ctx,
		`INSERT INTO video.videos (id, user_id, status, original_key) VALUES ($1::uuid, $2::uuid, 'PENDING', 'u/v.mp4')`,
		id, "22222222-2222-2222-2222-222222222222",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := repo.UpdateStatus(ctx, id, domain.StatusCompleted, "u/v.zip", ""); err != nil {
		t.Fatalf("update: %v", err)
	}

	status, err := repo.GetStatus(ctx, id)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status != domain.StatusCompleted {
		t.Fatalf("status = %s", status)
	}

	if _, err := repo.GetStatus(ctx, "00000000-0000-0000-0000-000000000000"); err != domain.ErrVideoNotFound {
		t.Fatalf("esperado ErrVideoNotFound, got %v", err)
	}
}

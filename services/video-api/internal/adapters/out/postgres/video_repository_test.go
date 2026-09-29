//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/fiapx/video-api/internal/domain"
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
		`CREATE INDEX IF NOT EXISTS idx_videos_user_id ON video.videos (user_id)`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}

	return pool
}

func TestVideoRepositoryLifecycle(t *testing.T) {
	repo := NewVideoRepository(newTestDB(t))
	ctx := context.Background()

	v := domain.Video{
		ID:          "11111111-1111-1111-1111-111111111111",
		UserID:      "22222222-2222-2222-2222-222222222222",
		Status:      domain.StatusPending,
		OriginalKey: "u/v.mp4",
	}

	created, err := repo.Create(ctx, v)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != domain.StatusPending || created.CreatedAt.IsZero() {
		t.Fatalf("dados inesperados: %+v", created)
	}

	if err := repo.UpdateStatus(ctx, v.ID, domain.StatusCompleted, "u/v.zip", ""); err != nil {
		t.Fatalf("update: %v", err)
	}

	found, err := repo.FindByID(ctx, v.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.Status != domain.StatusCompleted || found.ZipKey != "u/v.zip" {
		t.Fatalf("dados inesperados: %+v", found)
	}

	list, err := repo.ListByUser(ctx, v.UserID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != v.ID {
		t.Fatalf("list inesperada: %+v", list)
	}

	if _, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); err != domain.ErrVideoNotFound {
		t.Fatalf("esperado ErrVideoNotFound, got %v", err)
	}
}

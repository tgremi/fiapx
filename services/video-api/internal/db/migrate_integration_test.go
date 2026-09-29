//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRunMigrations(t *testing.T) {
	ctx := context.Background()

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
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

	if err := RunMigrations(connStr, "video_schema_migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM video.videos`).Scan(&n); err != nil {
		t.Fatalf("tabela video.videos não foi criada: %v", err)
	}
}

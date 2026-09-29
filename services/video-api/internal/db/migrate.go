package db

import (
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func RunMigrations(databaseURL, migrationsTable string) error {
	migrateURL := buildMigrationsURL(databaseURL, migrationsTable)

	src, err := iofs.New(MigrationsFS, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func buildMigrationsURL(databaseURL, migrationsTable string) string {
	u := "pgx5://" + strings.TrimPrefix(databaseURL, "postgres://")
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	return u + sep + "x-migrations-table=" + migrationsTable
}

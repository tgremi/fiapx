package db

import "testing"

func TestBuildMigrationsURL(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		table string
		want  string
	}{
		{
			name:  "com query string",
			url:   "postgres://u:p@host:5432/db?sslmode=disable",
			table: "auth_schema_migrations",
			want:  "pgx5://u:p@host:5432/db?sslmode=disable&x-migrations-table=auth_schema_migrations",
		},
		{
			name:  "sem query string",
			url:   "postgres://u:p@host:5432/db",
			table: "auth_schema_migrations",
			want:  "pgx5://u:p@host:5432/db?x-migrations-table=auth_schema_migrations",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildMigrationsURL(tc.url, tc.table); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

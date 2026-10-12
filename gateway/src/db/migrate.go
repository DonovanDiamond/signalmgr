package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every embedded migration not yet recorded in
// schema_migrations, in filename order, each inside its own transaction, and
// records it before moving on. It is safe to call on every startup: an
// already-applied file is skipped, and calling it again with nothing new to
// apply returns an empty, non-error slice.
func Migrate(ctx context.Context, h *sqlx.DB) ([]string, error) {
	if _, err := h.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return nil, fmt.Errorf("db: create schema_migrations: %w", err)
	}

	names, err := migrationNames()
	if err != nil {
		return nil, err
	}

	applied := map[string]bool{}
	var versions []string
	if err := h.SelectContext(ctx, &versions, `SELECT version FROM schema_migrations`); err != nil {
		return nil, fmt.Errorf("db: read schema_migrations: %w", err)
	}
	for _, v := range versions {
		applied[v] = true
	}

	var ran []string
	for _, name := range names {
		if applied[name] {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return ran, fmt.Errorf("db: read migration %s: %w", name, err)
		}

		tx, err := h.BeginTxx(ctx, nil)
		if err != nil {
			return ran, fmt.Errorf("db: begin %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			tx.Rollback()
			return ran, fmt.Errorf("db: apply %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return ran, fmt.Errorf("db: record %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return ran, fmt.Errorf("db: commit %s: %w", name, err)
		}

		ran = append(ran, name)
	}

	return ran, nil
}

// migrationNames returns the embedded *.sql filenames in filename order,
// which is also application order (001_, 002_, ...).
func migrationNames() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("db: list migrations: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

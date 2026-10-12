// Package db owns the SQLite connection, the migrations and every query.
package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	// ErrNotFound means no row matched.
	ErrNotFound = errors.New("not found")
	// ErrDuplicate means a unique constraint rejected the write.
	ErrDuplicate = errors.New("already exists")
)

// Open opens (creating if needed) the SQLite file at path, enables foreign
// keys, a busy timeout and WAL, and applies all migrations.
//
// "%", "?" and "#" in path are percent-escaped so path content cannot end the
// URI path early and inject a pragma of its own.
func Open(path string) (*sqlx.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("db: creating directory for %s: %w", path, err)
	}
	escaped := strings.NewReplacer("%", "%25", "?", "%3F", "#", "%23").Replace(path)
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", escaped)

	h, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", path, err)
	}
	if err := h.Ping(); err != nil {
		h.Close()
		return nil, fmt.Errorf("db: ping %s: %w", path, err)
	}
	if _, err := Migrate(context.Background(), h); err != nil {
		h.Close()
		return nil, err
	}
	return h, nil
}

// isDuplicate reports whether err is a SQLite primary-key or unique violation.
func isDuplicate(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	return se.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY || se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

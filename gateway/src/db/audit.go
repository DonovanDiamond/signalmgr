package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// AuditEntry is one mutating action. Rows are never updated or deleted.
type AuditEntry struct {
	ID     int64     `db:"id" json:"id"`
	At     time.Time `db:"at" json:"at"`
	Actor  string    `db:"actor" json:"actor"`
	Action string    `db:"action" json:"action"`
	Number string    `db:"number" json:"number"`
	Detail string    `db:"detail" json:"detail"`
	Result string    `db:"result" json:"result"`
}

// InsertAudit stores e; a zero At becomes now (UTC).
func InsertAudit(ctx context.Context, x *sqlx.DB, e AuditEntry) error {
	if e.At.IsZero() {
		e.At = time.Now()
	}
	e.At = e.At.UTC()
	_, err := x.NamedExecContext(ctx, `
		INSERT INTO audit (at, actor, action, number, detail, result)
		VALUES (:at, :actor, :action, :number, :detail, :result)`, e)
	if err != nil {
		return fmt.Errorf("db: inserting audit entry %q by %q: %w", e.Action, e.Actor, err)
	}
	return nil
}

// ListAudit returns up to limit entries, newest first.
func ListAudit(ctx context.Context, x *sqlx.DB, limit int) ([]AuditEntry, error) {
	if limit < 1 {
		return nil, fmt.Errorf("db: audit limit must be at least 1, got %d", limit)
	}
	out := []AuditEntry{}
	err := x.SelectContext(ctx, &out,
		`SELECT id, at, actor, action, number, detail, result FROM audit ORDER BY at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("db: listing audit: %w", err)
	}
	return out, nil
}

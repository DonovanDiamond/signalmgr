package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Number is a phone number the gateway manages. RegisterRequestedAt is set
// while a registration is awaiting its verification code.
type Number struct {
	Number              string     `db:"number"`
	Label               string     `db:"label"`
	RegisterRequestedAt *time.Time `db:"register_requested_at"`
	CreatedAt           time.Time  `db:"created_at"`
}

const numberColumns = `number, label, register_requested_at, created_at`

func ListNumbers(ctx context.Context, x *sqlx.DB) ([]Number, error) {
	out := []Number{}
	if err := x.SelectContext(ctx, &out, `SELECT `+numberColumns+` FROM numbers ORDER BY number`); err != nil {
		return nil, fmt.Errorf("db: listing numbers: %w", err)
	}
	return out, nil
}

func GetNumber(ctx context.Context, x *sqlx.DB, number string) (Number, error) {
	var n Number
	err := x.GetContext(ctx, &n, `SELECT `+numberColumns+` FROM numbers WHERE number = ?`, number)
	if errors.Is(err, sql.ErrNoRows) {
		return Number{}, ErrNotFound
	}
	if err != nil {
		return Number{}, fmt.Errorf("db: getting number %s: %w", number, err)
	}
	return n, nil
}

func InsertNumber(ctx context.Context, x *sqlx.DB, number, label string) error {
	_, err := x.ExecContext(ctx, `INSERT INTO numbers (number, label, created_at) VALUES (?, ?, ?)`,
		number, label, time.Now().UTC())
	if isDuplicate(err) {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("db: inserting number %s: %w", number, err)
	}
	return nil
}

// UpsertNumberLabel sets the label, creating the row if missing.
func UpsertNumberLabel(ctx context.Context, x *sqlx.DB, number, label string) error {
	_, err := x.ExecContext(ctx, `
		INSERT INTO numbers (number, label, created_at) VALUES (?, ?, ?)
		ON CONFLICT (number) DO UPDATE SET label = excluded.label`,
		number, label, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("db: setting label of %s: %w", number, err)
	}
	return nil
}

// SetRegisterRequested records when a registration was requested; nil clears
// it. Creates the row if missing.
func SetRegisterRequested(ctx context.Context, x *sqlx.DB, number string, at *time.Time) error {
	if at != nil {
		u := at.UTC()
		at = &u
	}
	_, err := x.ExecContext(ctx, `
		INSERT INTO numbers (number, register_requested_at, created_at) VALUES (?, ?, ?)
		ON CONFLICT (number) DO UPDATE SET register_requested_at = excluded.register_requested_at`,
		number, at, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("db: setting register_requested_at of %s: %w", number, err)
	}
	return nil
}

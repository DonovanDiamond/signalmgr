package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	SourceLocal = "local"
	SourceAD    = "ad"
)

// User is an account. PasswordHash is nil for AD accounts.
type User struct {
	ID           int64     `db:"id"`
	Username     string    `db:"username"`
	Name         string    `db:"name"`
	Role         string    `db:"role"`
	Source       string    `db:"source"`
	PasswordHash *string   `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

// Session is one issued token; ID is the JWT "jti".
type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
}

const userColumns = `id, username, name, role, source, password_hash, created_at`

func CountUsers(ctx context.Context, x *sqlx.DB) (int, error) {
	var n int
	if err := x.GetContext(ctx, &n, `SELECT COUNT(*) FROM users`); err != nil {
		return 0, fmt.Errorf("db: counting users: %w", err)
	}
	return n, nil
}

func GetUserByUsername(ctx context.Context, x *sqlx.DB, username string) (User, error) {
	var u User
	err := x.GetContext(ctx, &u, `SELECT `+userColumns+` FROM users WHERE username = ?`, username)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("db: getting user %q: %w", username, err)
	}
	return u, nil
}

// InsertLocalUser creates a local account with an already-hashed password.
func InsertLocalUser(ctx context.Context, x *sqlx.DB, username, name, role, passwordHash string) error {
	_, err := x.ExecContext(ctx,
		`INSERT INTO users (username, name, role, source, password_hash, created_at) VALUES (?, ?, ?, 'local', ?, ?)`,
		username, name, role, passwordHash, time.Now().UTC())
	if isDuplicate(err) {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("db: inserting user %q: %w", username, err)
	}
	return nil
}

// UpsertADUser creates the directory account or refreshes its name and role
// from the directory. A local account with the same username is never
// touched: that returns ErrDuplicate.
func UpsertADUser(ctx context.Context, x *sqlx.DB, username, name, role string) (User, error) {
	_, err := x.ExecContext(ctx, `
		INSERT INTO users (username, name, role, source, created_at) VALUES (?, ?, ?, 'ad', ?)
		ON CONFLICT (username) DO UPDATE SET name = excluded.name, role = excluded.role
		WHERE users.source = 'ad'`,
		username, name, role, time.Now().UTC())
	if err != nil {
		return User{}, fmt.Errorf("db: upserting directory user %q: %w", username, err)
	}
	u, err := GetUserByUsername(ctx, x, username)
	if err != nil {
		return User{}, err
	}
	if u.Source != SourceAD {
		return User{}, ErrDuplicate
	}
	return u, nil
}

func InsertSession(ctx context.Context, x *sqlx.DB, s Session) error {
	_, err := x.ExecContext(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		s.ID, s.UserID, s.ExpiresAt.UTC())
	if err != nil {
		return fmt.Errorf("db: inserting session for user %d: %w", s.UserID, err)
	}
	return nil
}

// GetSessionUser returns the user a session belongs to, or ErrNotFound if the
// session is unknown, revoked or expired at now.
func GetSessionUser(ctx context.Context, x *sqlx.DB, id string, now time.Time) (User, error) {
	var u User
	err := x.GetContext(ctx, &u, `
		SELECT u.id, u.username, u.name, u.role, u.source, u.password_hash, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.revoked_at IS NULL AND s.expires_at > ?`, id, now.UTC())
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("db: getting session: %w", err)
	}
	return u, nil
}

func RevokeSession(ctx context.Context, x *sqlx.DB, id string, now time.Time) error {
	_, err := x.ExecContext(ctx, `UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now.UTC(), id)
	if err != nil {
		return fmt.Errorf("db: revoking session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions drops sessions that expired before the given time;
// they can no longer verify, so nothing depends on them.
func DeleteExpiredSessions(ctx context.Context, x *sqlx.DB, before time.Time) error {
	_, err := x.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, before.UTC())
	if err != nil {
		return fmt.Errorf("db: deleting expired sessions: %w", err)
	}
	return nil
}

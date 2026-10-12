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
	Disabled     bool      `db:"disabled"`
	CreatedAt    time.Time `db:"created_at"`
}

// Session is one issued access token; ID is the JWT "jti". RefreshHash is the
// SHA-256 hex digest of the opaque refresh token, never the token itself.
type Session struct {
	ID               string
	UserID           int64
	ExpiresAt        time.Time
	RefreshHash      string
	RefreshExpiresAt time.Time
}

const userColumns = `id, username, name, role, source, password_hash, disabled, created_at`

func CountUsers(ctx context.Context, x *sqlx.DB) (int, error) {
	var n int
	if err := x.GetContext(ctx, &n, `SELECT COUNT(*) FROM users`); err != nil {
		return 0, fmt.Errorf("db: counting users: %w", err)
	}
	return n, nil
}

func GetUserByID(ctx context.Context, x *sqlx.DB, id int64) (User, error) {
	var u User
	err := x.GetContext(ctx, &u, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("db: getting user %d: %w", id, err)
	}
	return u, nil
}

// ListUsers returns every account, ordered by username.
func ListUsers(ctx context.Context, x *sqlx.DB) ([]User, error) {
	us := []User{}
	if err := x.SelectContext(ctx, &us, `SELECT `+userColumns+` FROM users ORDER BY username`); err != nil {
		return nil, fmt.Errorf("db: listing users: %w", err)
	}
	return us, nil
}

// SetUserDisabled returns ErrNotFound if there is no such user.
func SetUserDisabled(ctx context.Context, x *sqlx.DB, username string, disabled bool) error {
	return requireOneRow(x.ExecContext(ctx, `UPDATE users SET disabled = ? WHERE username = ?`, disabled, username))
}

// SetUserPassword replaces a local account's hash; ErrNotFound if there is
// no such local user.
func SetUserPassword(ctx context.Context, x *sqlx.DB, username, passwordHash string) error {
	return requireOneRow(x.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE username = ? AND source = 'local'`, passwordHash, username))
}

func requireOneRow(res sql.Result, err error) error {
	if err != nil {
		return fmt.Errorf("db: updating user: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
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
	_, err := x.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, refresh_hash, refresh_expires_at) VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.ExpiresAt.UTC(), s.RefreshHash, s.RefreshExpiresAt.UTC())
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
		SELECT u.id, u.username, u.name, u.role, u.source, u.password_hash, u.disabled, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.revoked_at IS NULL AND s.expires_at > ? AND u.disabled = 0`, id, now.UTC())
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

// ClaimRefresh revokes the live session holding refreshHash and returns its
// user's id and refresh deadline, in one statement so a token can be spent only once even under
// concurrent requests. ErrNotFound if the token is unknown, spent, revoked or expired.
func ClaimRefresh(ctx context.Context, x *sqlx.DB, refreshHash string, now time.Time) (int64, time.Time, error) {
	var claimed struct {
		UserID           int64     `db:"user_id"`
		RefreshExpiresAt time.Time `db:"refresh_expires_at"`
	}
	err := x.GetContext(ctx, &claimed, `
		UPDATE sessions SET revoked_at = ?
		WHERE refresh_hash = ? AND revoked_at IS NULL AND refresh_expires_at > ?
		RETURNING user_id, refresh_expires_at`, now.UTC(), refreshHash, now.UTC())
	if errors.Is(err, sql.ErrNoRows) {
		return 0, time.Time{}, ErrNotFound
	}
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("db: claiming refresh token: %w", err)
	}
	return claimed.UserID, claimed.RefreshExpiresAt, nil
}

// RevokeUserSessions revokes every live session of the user.
func RevokeUserSessions(ctx context.Context, x *sqlx.DB, userID int64, now time.Time) error {
	_, err := x.ExecContext(ctx, `UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now.UTC(), userID)
	if err != nil {
		return fmt.Errorf("db: revoking sessions of user %d: %w", userID, err)
	}
	return nil
}

// DeleteExpiredSessions drops sessions that can neither verify nor be
// refreshed any more as of the given time.
func DeleteExpiredSessions(ctx context.Context, x *sqlx.DB, before time.Time) error {
	_, err := x.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE expires_at < ? AND (refresh_expires_at IS NULL OR refresh_expires_at < ?)`, before.UTC(), before.UTC())
	if err != nil {
		return fmt.Errorf("db: deleting expired sessions: %w", err)
	}
	return nil
}

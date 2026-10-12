-- A disabled user cannot sign in or refresh; users are never deleted.
ALTER TABLE users ADD COLUMN disabled INTEGER NOT NULL DEFAULT 0 CHECK (disabled IN (0, 1));

-- sessions.expires_at is now the short access-token expiry. The opaque refresh
-- token is stored only as a SHA-256 hex digest; NULL on sessions that predate
-- refresh tokens, which therefore cannot be refreshed. Rotating a refresh token
-- revokes its session and inserts a new one.
ALTER TABLE sessions ADD COLUMN refresh_hash TEXT;
ALTER TABLE sessions ADD COLUMN refresh_expires_at TIMESTAMP;  -- UTC
CREATE UNIQUE INDEX sessions_refresh ON sessions (refresh_hash);

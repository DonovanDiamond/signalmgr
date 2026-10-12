-- numbers holds what signal-cli-rest-api cannot: our label and the "awaiting
-- code" fact. A row may exist for a number Signal does not know yet.
CREATE TABLE numbers (
    number                TEXT PRIMARY KEY CHECK (number GLOB '+[0-9]*'),
    label                 TEXT NOT NULL DEFAULT '',
    register_requested_at TIMESTAMP,          -- NULL = no registration pending
    created_at            TIMESTAMP NOT NULL  -- UTC
);

-- Append-only: rows are never updated or deleted.
CREATE TABLE audit (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    at     TIMESTAMP NOT NULL,                -- UTC
    actor  TEXT NOT NULL,
    action TEXT NOT NULL,
    number TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL
);
CREATE INDEX audit_at ON audit (at DESC, id DESC);

CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    source        TEXT NOT NULL CHECK (source IN ('local', 'ad')),
    password_hash TEXT,                       -- NULL for AD accounts
    created_at    TIMESTAMP NOT NULL,         -- UTC
    CHECK ((source = 'local') = (password_hash IS NOT NULL))
);

-- One row per issued token (id = the JWT "jti"); revoking sets revoked_at.
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users (id),
    expires_at TIMESTAMP NOT NULL,            -- UTC
    revoked_at TIMESTAMP
);
CREATE INDEX sessions_user ON sessions (user_id);

-- An account without a phone number is keyed by its ACI (a UUID), so the
-- number column now holds either an E.164 number or an ACI. SQLite cannot
-- alter a CHECK, hence the rebuild.
CREATE TABLE numbers_new (
    number                TEXT PRIMARY KEY CHECK (
        number GLOB '+[0-9]*'
        OR (length(number) = 36 AND number NOT GLOB '*[^0-9a-f-]*')),
    label                 TEXT NOT NULL DEFAULT '',
    register_requested_at TIMESTAMP,
    created_at            TIMESTAMP NOT NULL
);
INSERT INTO numbers_new SELECT number, label, register_requested_at, created_at FROM numbers;
DROP TABLE numbers;
ALTER TABLE numbers_new RENAME TO numbers;

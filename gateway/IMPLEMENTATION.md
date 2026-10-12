# Implementation

See also [README.md](README.md) (what it is) and [QUESTIONS.md](QUESTIONS.md) (why).

Go module `github.com/DonovanDiamond/signalmgr/gateway`, nested in the signalmgr repository with `replace github.com/DonovanDiamond/signalmgr => ../`, so fiber, sqlx and the rest stay out of the library's own `go.mod`.

## Layout

```
cmd/gateway/main.go   wiring only: flag -config, zerolog, graceful shutdown (10 s)
cmd/fakesignal/       dev server around src/fakesignal
src/config/           config.yaml load, defaults, validation (unknown keys refused)
src/db/               sqlx + pure-Go SQLite (modernc), embedded migrations, every query
src/auth/             local + AD sign-in, EdDSA JWT issue/verify, sessions in the database
src/signal/           service over the signalmgr library: validation, state merge, audit
src/api/              fiber v3 routes, JWT middleware, error mapping
src/fakesignal/       in-memory signal-cli-rest-api used by tests and cmd/fakesignal
web/                  SvelteKit (adapter-static); web/embed.go embeds web/build
```

Dependencies point inward: `api` calls `auth` and `signal`; `signal` and `auth` call `db`; `db` calls neither.

## Database (SQLite, `src/db/migrations`)

- `numbers(number PK, label, register_requested_at NULL, created_at)`: what Signal cannot hold. A row may exist for a number Signal does not know.
- `audit(id, at, actor, action, number, detail, result)`: append-only.
- `users(id, username UNIQUE, name, role admin|user, source local|ad, password_hash, created_at)`: a hash exists exactly when the source is local.
- `sessions(id = JWT jti, user_id, expires_at, revoked_at)`.

State of a number: in Signal's account list means `registered`; else `register_requested_at` set means `awaiting_code`; else `not_registered`.

## HTTP API

JSON under `/api`; auth is `Authorization: Bearer <jwt>` except login. Bodies are limited to 16 KiB. Success with no content is `204`. Errors are `{"error": "..."}`.

| Route | Role | Body / result |
|---|---|---|
| `POST /api/auth/login` | none | `{username,password}` gives `{token,user:{username,name,role}}`; 10 failures per minute per IP, then 429 |
| `POST /api/auth/logout` | any | revokes the session; 204 |
| `GET /api/auth/me` | any | the user |
| `GET /api/gateway` | any | `{online,version,mode,installationName}`; `online:false` when Signal does not answer |
| `GET /api/numbers` | any | `[{number,label,state,devices:[{id,name,created,lastSeen,primary}]}]`; 502 when Signal is unreachable |
| `POST /api/numbers` | admin | `{number,label}` |
| `PATCH /api/numbers/:number` | admin | `{label}` |
| `POST /api/numbers/:number/register` | admin | `{captcha,voice}` |
| `POST /api/numbers/:number/verify` | admin | `{code,pin?}` |
| `POST /api/numbers/:number/unregister` | admin | `{deleteAccount,deleteLocalData}` |
| `DELETE /api/numbers/:number/devices/:id` | admin | id 2 or higher |
| `PUT /api/numbers/:number/pin`, `DELETE ...` | admin | `{pin}` |
| `POST /api/numbers/:number/rate-limit` | admin | `{challengeToken,captcha}` |
| `GET /api/link/qr?deviceName=` | admin | `image/png` |
| `GET /api/link/uri?deviceName=` | admin | `{uri}` |
| `GET /api/audit?limit=` | admin | newest first, default 200, at most 1000 |

`:number` is URL-escaped (`%2B27...`); the app runs with `UnescapePath`. Any other `/api` path is a JSON 404; any other path serves the UI, falling back to `index.html` for client-side routes.

Link routes are admin-only because linking a device gives that device full access to the account.

### Error mapping (`src/api/server.go`, `fail`)

| Cause | Status |
|---|---|
| `signal.ErrInvalid`, malformed JSON | 400 |
| `auth.ErrInvalidCredentials`, `ErrInvalidToken`, missing token | 401 |
| role too low, `auth.ErrNoRole` | 403 |
| `db.ErrNotFound` | 404 |
| `db.ErrDuplicate` | 409 |
| body over 16 KiB | 413 |
| `*signalmgr.APIError` | its own `Status` and `Message` only |
| `signal.ErrUnreachable` | 502 |
| `auth.ErrDirectoryUnreachable` | 503 |
| anything else | 500 `internal error`, logged |

`APIError.Error()` is never sent or logged: it contains the request path, which for verify includes the code.

## Tests

`go test ./...` uses temp SQLite files and `httptest` around `src/fakesignal`. The signalmgr library keeps its API URL in a package global, so tests that start a signal service must not run in parallel. `src/api` covers login, 401/403 per route, add-number validation, the register to verify flow, upstream 429 and unreachable passthrough, the body limit, the login limiter and the SPA fallback.

## Build

`make build-linux` / `make build-windows` run `npm ci && npm run build` in `web/`, then `CGO_ENABLED=0 go build` into `dist/`.

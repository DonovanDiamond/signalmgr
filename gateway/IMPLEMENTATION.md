# Implementation

See also [README.md](README.md) (what it is) and [QUESTIONS.md](QUESTIONS.md) (why).

Go module `github.com/DonovanDiamond/signalmgr/gateway`, nested in the signalmgr repository, so fiber, sqlx and the rest stay out of the library's own `go.mod`.

It requires the library at its own release version (`gateway/vX.Y.Z` needs `signalmgr vX.Y.Z`) and has no `replace`. The root `go.work` (`use . ./gateway`) points local builds at the checkout's library source; `GOWORK=off` builds against the pinned release. The root `Makefile` (`make release`, `make image`) keeps the two tags and the image tag in step; the binary's `version` variable is set with `-ldflags "-X main.version=..."` (default `dev`).

## Layout

```
cmd/gateway/main.go   wiring only: flag -config, zerolog, graceful shutdown (10 s)
cmd/fakesignal/       dev server around src/fakesignal (-seed design, -link-after)
cmd/devseed/          dev tool: design labels into an empty gateway db
src/config/           config.yaml load, Defaults/Resolve, validation (unknown keys refused)
src/db/               sqlx + pure-Go SQLite (modernc), embedded migrations, every query
src/auth/             local + AD sign-in, EdDSA JWT issue/verify, refresh tokens, user management, sessions in the database
src/signal/           service over the signalmgr library: validation, state merge, audit
src/api/              fiber v3 routes, JWT middleware, error mapping
src/fakesignal/       in-memory signal-cli-rest-api used by tests and cmd/fakesignal
web/                  SvelteKit (adapter-static); web/embed.go embeds web/build; web/notbuilt is the fallback page
```

Dependencies point inward: `api` calls `auth` and `signal`; `signal` and `auth` call `db`; `db` calls neither.

## Database (SQLite, `src/db/migrations`)

- `numbers(number PK, label, register_requested_at NULL, created_at)`: what Signal cannot hold. A row may exist for a number Signal does not know.
- `audit(id, at, actor, action, number, detail, result)`: append-only.
- `users(id, username UNIQUE, name, role admin|user, source local|ad, password_hash, disabled, created_at)`: a hash exists exactly when the source is local.
- `sessions(id = JWT jti, user_id, expires_at, revoked_at, refresh_hash UNIQUE, refresh_expires_at)`: one row per issued access token. `expires_at` is the access expiry; `refresh_hash` is the SHA-256 hex of the opaque refresh token (NULL for sessions from before migration 002, which cannot refresh). Refreshing revokes the old row and inserts a new one.

State of a number: in Signal's account list means `registered`; else `register_requested_at` set means `awaiting_code`; else `not_registered`.

## HTTP API

JSON under `/api`; auth is `Authorization: Bearer <access jwt>` except login and refresh. Bodies are limited to 16 KiB. Success with no content is `204`. Errors are `{"error": "..."}`.

| Route | Role | Body / result |
|---|---|---|
| `POST /api/auth/login` | none | `{username,password}` gives `{token,refreshToken,user:{username,name,role,source}}`; 10 failures per minute per IP, then 429 |
| `POST /api/auth/refresh` | none | `{refreshToken}` gives the same body as login; the token is spent. Its own 10-failures-per-minute counter. Unknown, spent, expired, revoked or disabled-user all give 401 |
| `POST /api/auth/logout` | any | revokes the session; 204 |
| `GET /api/auth/me` | any | the user |
| `POST /api/auth/password` | any | `{current,new}`; local accounts only; 204 |
| `GET /api/users` | admin | `[{username,name,role,source,disabled}]` |
| `POST /api/users` | admin | `{username,name,role,password}` creates a local user |
| `PATCH /api/users/:username` | admin | `{disabled}` (required); local users other than yourself |
| `PUT /api/users/:username/password` | admin | `{password}`; local users only; ends that user's sessions |
| `GET /api/gateway` | any | `{online,version,mode,installationName}`; `online:false` when Signal does not answer |
| `GET /api/numbers` | any | `[{number,label,state,devices:[{id,name,created,lastSeen,primary}]}]`; 502 when Signal is unreachable |
| `POST /api/numbers` | admin | `{number,label}` |
| `PATCH /api/numbers/:number` | admin | `{label}` |
| `POST /api/numbers/:number/register` | admin | `{captcha,voice}` |
| `POST /api/numbers/:number/verify` | admin | `{code,pin?}` |
| `POST /api/numbers/:number/unregister` | admin | `{deleteAccount,deleteLocalData}` |
| `POST /api/numbers/:number/devices` | admin | `{uri}` (`sgnl://linkdevice?...`, at most 2048 characters); adds a device to the number; the URI is not audited |
| `DELETE /api/numbers/:number/devices/:id` | admin | id 2 or higher |
| `PUT /api/numbers/:number/pin`, `DELETE ...` | admin | `{pin}` |
| `POST /api/numbers/:number/rate-limit` | admin | `{challengeToken,captcha}` |
| `GET /api/link/qr?deviceName=` | admin | `image/png` |
| `GET /api/link/uri?deviceName=` | admin | `{uri}` |
| `GET /api/audit?limit=` | admin | newest first, default 200, at most 1000 |

`:number` is URL-escaped (`%2B27...`); the app runs with `UnescapePath`. Any other `/api` path is a JSON 404; any other path serves the UI, falling back to `index.html` for client-side routes.

Link routes are admin-only because linking a device gives that device full access to the account.

User input limits (`src/auth/users.go`): username `[A-Za-z0-9._@-]{1,64}`, name 1 to 100 characters, password 12 to 72 bytes (bcrypt's input limit), role `admin` or `user`. A wrong current password on `POST /auth/password` is a 400, not a 401, so the client does not discard a valid session.

### Error mapping (`src/api/server.go`, `fail`)

| Cause | Status |
|---|---|
| `signal.ErrInvalid`, `auth.ErrInvalidInput`, malformed JSON | 400 |
| `auth.ErrInvalidCredentials`, `ErrInvalidToken`, missing token | 401 |
| role too low, `auth.ErrNoRole`, `auth.ErrDisabled` | 403 |
| `db.ErrNotFound` | 404 |
| `db.ErrDuplicate` | 409 |
| body over 16 KiB | 413 |
| `*signalmgr.APIError` | its own `Status` and `Message` only |
| `signal.ErrUnreachable` | 502 |
| `auth.ErrDirectoryUnreachable`, `*auth.DirectoryTimeoutError` (message names the `ad.timeout`) | 503 |
| anything else | 500 `internal error`, logged |

`APIError.Error()` is never sent or logged: it contains the request path, which for verify includes the code.

## Tests

`go test ./...` uses temp SQLite files and `httptest` around `src/fakesignal`. `web.FS()` and `config.Defaults().Resolve()` are covered by `web/embed_test.go` and `src/config`; refresh rotation and user management by `src/auth/refresh_test.go` and `auth_test.go`. The signalmgr library keeps its API URL in a package global, so tests that start a signal service must not run in parallel. `src/api` covers login, 401/403 per route, add-number validation, the register to verify flow, upstream 429 and unreachable passthrough, the body limit, the login limiter and the SPA fallback.

## Configuration notes

- `jwt.access_ttl` (default 15m) and `jwt.refresh_ttl` (default 8h, must exceed the access TTL). A refresh carries the original refresh expiry forward unchanged, so `refresh_ttl` is an absolute limit from sign-in, not an idle one. `jwt.token_ttl` is refused as an unknown key.
- `trusted_proxies`: CIDRs only. When set, fiber trusts `X-Forwarded-For` from those peers; otherwise the header is ignored.
- `ad.timeout` (default 10s). The LDAP library takes no deadline, so on timeout the call is abandoned on its goroutine and the sign-in fails with 503.
- `config.Defaults()` is the configuration with no file; `Resolve()` applies the `SIGNALGW_ADMIN_*` overrides and validates. `cmd/gateway` uses `Defaults().Resolve()` for a missing file.

## Build

`make build-linux` / `make build-windows` run `npm ci && npm run build` in `web/`, then `CGO_ENABLED=0 go build` into `dist/`.

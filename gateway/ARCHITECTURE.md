# gateway — Architecture

**What:** Web app and HTTP API for managing Signal numbers (register, verify, unregister, linked devices, PIN, rate-limit challenge) on top of the `signalmgr` library. Embeds a SvelteKit UI in a single Go binary.
**Module:** `github.com/DonovanDiamond/signalmgr/gateway` — Go 1.25.5 (see `go.mod`)
**Upstream:** signal-cli-rest-api via the `signalmgr` library (`signalmgr.API_URL` set from `signal_api_url`)
**Deps:** `gofiber/fiber/v3`, `jmoiron/sqlx` with `modernc.org/sqlite` (pure-Go SQLite), `rs/zerolog`, `golang-jwt/jwt/v5`, `DonovanDiamond/go-ad-auth/v3` (Active Directory auth), `gopkg.in/yaml.v3`; UI: SvelteKit (adapter-static) + Svelte 5 + Tailwind 4/daisyUI 5 + `@fontsource/archivo` and `archivo-narrow`, Vite, Playwright

## Layout
| Path | Role |
|---|---|
| `cmd/gateway/main.go` | Entrypoint: `-config` flag, `version` var (set by `-X main.version`), wires config → db → auth → signal service → api, graceful shutdown |
| `cmd/fakesignal/main.go` | Dev entrypoint: serves the fake signal-cli-rest-api (`-listen`, `-empty`, `-seed design`, `-accounts-format strings|objects`, `-link-after`) |
| `cmd/devseed/main.go` | Dev tool: writes the design's number labels into an empty gateway database (`-db`) |
| `src/config` | `config.yaml` loading, exported `Defaults()` / `Resolve()`, env overrides, validation |
| `src/db` | SQLite open + embedded migrations; numbers, audit, users, sessions queries (sessions carry refresh-token hashes) |
| `src/auth` | Local + Active Directory login, JWT issue/verify/revoke, refresh-token rotation, user management, signing keys, first-admin seed |
| `src/signal` | Service over `signalmgr`: merges upstream accounts with db labels/state, validates input, audits mutations |
| `src/fakesignal` | In-memory fake of the signal-cli-rest-api endpoints the gateway uses |
| `src/api` | Fiber app: routes, auth/role middleware, error mapping, body limit, per-route failure rate limit, trusted proxies, SPA serving |
| `web/` | SvelteKit static UI; `web/embed.go` embeds `web/build` (`web.FS()`); `web/notbuilt/` is the placeholder page served when there is no build; `web/build/.gitkeep` keeps the embed directory present in a clean checkout |
| `../go.work`, `../Makefile` | Workspace (`use . ./gateway`) for local builds against the library source; root targets `image` and `release` build and tag the gateway at the shared version (`gateway/go.mod` requires the same `signalmgr` version, no `replace`) |
| `config.example.yaml`, `Makefile` | Sample config; targets `web`, `build-linux`, `build-windows`, `test`, `fakesignal`, `run`, `clean` |

## Flow
browser (`web/src/lib/api.ts` `apiFetch`, bearer access token; on a 401 it spends the refresh token once via `session.refresh` and retries) → `/api/*` (`src/api/server.go`) → `requireAuth` (`auth.Service.Verify`: JWT + live session row) → `requireAdmin` where the route needs it → handler (`handlers.go`) → `signal.Service` (validate, call `signalmgr`, write audit) → signal-cli-rest-api.
- db holds what upstream cannot: number labels, "awaiting code" state, audit log, users, sessions.
- `signal.Service.Numbers` merges `signalmgr.GetAccounts` with `numbers` rows to derive state (`registered`, `awaiting_code`, `not_registered`). Accounts are keyed by id: the number, or the lowercase ACI for a numberless account; `signal.ParseID` checks path ids (E.164 or ACI).
- Sign-in returns a short-lived access JWT plus an opaque refresh token. `POST /auth/refresh` spends the refresh token (its session is revoked in one statement) and issues a fresh pair under a new session; only the SHA-256 digest of a refresh token is stored.
- Errors: `api.fail` maps `auth.Err*` and `signal.Err*` to HTTP statuses; unknown `/api/*` paths return 404 JSON; any other path serves the SPA (`index.html` fallback). Without a built UI `web.FS()` returns the placeholder page together with an error; `cmd/gateway` logs a warning and the API still serves.

## Roles
| Role | Source | Can |
|---|---|---|
| `admin` | local user or AD group map | everything |
| `user` | local user or AD group map | sign in, list numbers, read gateway info, change own password (local only) |
Role is enforced per route in `src/api/server.go`, never inside handlers. AD users in no mapped group cannot sign in (`auth.ErrNoRole`). An AD account cannot shadow a local one. A disabled local user cannot sign in (`auth.ErrDisabled`, 403) or refresh; AD users cannot be disabled here.

## HTTP routes (all under `/api`)
| Route | Auth | Handler |
|---|---|---|
| `POST /auth/login`, `POST /auth/refresh` | none, each rate limited per minute on failures | `login`, `refresh` |
| `POST /auth/logout`, `GET /auth/me` | any signed-in | `logout`, `me` |
| `POST /auth/password` | any signed-in (local accounts) | `changePassword` |
| `GET /users`, `POST /users` | admin | `users`, `addUser` |
| `PATCH /users/:username`, `PUT /users/:username/password` | admin | `patchUser` (disable/enable), `resetPassword` |
| `GET /gateway`, `GET /numbers` | any signed-in | `gateway`, `numbers` |
| `GET /audit` | admin | `audit` |
| `POST /numbers`, `PATCH /numbers/:number` | admin | `addNumber`, `setLabel` |
| `POST /numbers/:number/{register,verify,unregister,rate-limit}` | admin | `register`, `verify`, `unregister`, `rateLimit` |
| `POST /numbers/:number/devices` | admin | `linkDevice` (add a device to this number from a `sgnl://linkdevice?` URI) |
| `DELETE /numbers/:number/devices/:id` | admin | `removeDevice` |
| `PUT`/`DELETE /numbers/:number/pin` | admin | `setPin`, `removePin` |
| `GET /link/qr`, `GET /link/uri` | admin | `linkQR`, `linkURI` |

## Data (`src/db/migrations/`)
`001_initial.sql`: `numbers` (number, label, register_requested_at), `audit` (append-only), `users` (local with password hash, or `ad`), `sessions` (id = JWT `jti`, expires_at, revoked_at).
`002_refresh_and_disable.sql`: `users.disabled`; `sessions.refresh_hash` (unique) and `refresh_expires_at`. `sessions.expires_at` is now the access-token expiry; sessions from before the migration have no refresh hash and cannot be refreshed.
`003_numbers_aci_ids.sql`: rebuilds `numbers` so the key may be an ACI (lowercase UUID) as well as an E.164 number.

## Conventions
- Every mutating `signal.Service` method takes `actor` and writes one audit row (result `ok` or error text).
- Config unknown keys are refused; secrets are never echoed in errors.
- Timestamps stored UTC; display timezone from `timezone` config.
- Query helpers take `ctx` and `*sqlx.DB` as first params; `db.ErrNotFound` / `db.ErrDuplicate` are the only db sentinel errors.
- Web: Svelte 5 runes; `*.svelte.ts` hold shared state; each action dialog is a component mounted only while open, wrapping `Modal`. Tokens are kept in `sessionStorage`, not `localStorage`.

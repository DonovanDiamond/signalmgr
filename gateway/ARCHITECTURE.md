# gateway — Architecture

**What:** Web app and HTTP API for managing Signal numbers (register, verify, unregister, linked devices, PIN, rate-limit challenge) on top of the `signalmgr` library. Embeds a SvelteKit UI in a single Go binary.
**Module:** `github.com/DonovanDiamond/signalmgr/gateway` — Go 1.25.5 (see `go.mod`)
**Upstream:** signal-cli-rest-api via the `signalmgr` library (`signalmgr.API_URL` set from `signal_api_url`)
**Deps:** `gofiber/fiber/v3`, `jmoiron/sqlx` with `modernc.org/sqlite` (pure-Go SQLite), `rs/zerolog`, `golang-jwt/jwt/v5`, `DonovanDiamond/go-ad-auth/v3` (Active Directory auth), `gopkg.in/yaml.v3`; UI: SvelteKit (adapter-static) + Svelte 5 + Tailwind 4/daisyUI 5 + `@fontsource/archivo` and `archivo-narrow`, Vite, Playwright

## Layout
| Path | Role |
|---|---|
| `cmd/gateway/main.go` | Entrypoint: `-config` flag, wires config → db → auth → signal service → api, graceful shutdown |
| `cmd/fakesignal/main.go` | Dev entrypoint: serves the fake signal-cli-rest-api (`-listen`, `-empty`) |
| `src/config` | `config.yaml` loading, defaults, env overrides, validation |
| `src/db` | SQLite open + embedded migrations; numbers, audit, users, sessions queries |
| `src/auth` | Local + Active Directory login, JWT issue/verify/revoke, signing keys, first-admin seed |
| `src/signal` | Service over `signalmgr`: merges upstream accounts with db labels/state, validates input, audits mutations |
| `src/fakesignal` | In-memory fake of the signal-cli-rest-api endpoints the gateway uses |
| `src/api` | Fiber app: routes, auth/role middleware, error mapping, body limit, login rate limit, SPA serving |
| `web/` | SvelteKit static UI; `web/embed.go` embeds `web/build` (`web.FS()`) |
| `config.example.yaml`, `Makefile` | Sample config; targets `web`, `build-linux`, `build-windows`, `test`, `fakesignal`, `run`, `clean` |

## Flow
browser (`web/src/lib/api.ts` `apiFetch`, bearer token) → `/api/*` (`src/api/server.go`) → `requireAuth` (`auth.Service.Verify`: JWT + live session row) → `requireAdmin` where the route needs it → handler (`handlers.go`) → `signal.Service` (validate, call `signalmgr`, write audit) → signal-cli-rest-api.
- db holds what upstream cannot: number labels, "awaiting code" state, audit log, users, sessions.
- `signal.Service.Numbers` merges `signalmgr.GetAccounts` with `numbers` rows to derive state (`registered`, `awaiting_code`, `not_registered`).
- Errors: `api.fail` maps `auth.Err*` and `signal.Err*` to HTTP statuses; unknown `/api/*` paths return 404 JSON; any other path serves the SPA (`index.html` fallback). Without a built UI the API still serves.

## Roles
| Role | Source | Can |
|---|---|---|
| `admin` | local user or AD group map | everything |
| `user` | local user or AD group map | sign in, list numbers, read gateway info |
Role is enforced per route in `src/api/server.go`, never inside handlers. AD users in no mapped group cannot sign in (`auth.ErrNoRole`). An AD account cannot shadow a local one.

## HTTP routes (all under `/api`)
| Route | Auth | Handler |
|---|---|---|
| `POST /auth/login` | none, rate limited per minute on failures | `login` |
| `POST /auth/logout`, `GET /auth/me` | any signed-in | `logout`, `me` |
| `GET /gateway`, `GET /numbers` | any signed-in | `gateway`, `numbers` |
| `GET /audit` | admin | `audit` |
| `POST /numbers`, `PATCH /numbers/:number` | admin | `addNumber`, `setLabel` |
| `POST /numbers/:number/{register,verify,unregister,rate-limit}` | admin | `register`, `verify`, `unregister`, `rateLimit` |
| `DELETE /numbers/:number/devices/:id` | admin | `removeDevice` |
| `PUT`/`DELETE /numbers/:number/pin` | admin | `setPin`, `removePin` |
| `GET /link/qr`, `GET /link/uri` | admin | `linkQR`, `linkURI` |

## Data (`src/db/migrations/001_initial.sql`)
`numbers` (number, label, register_requested_at), `audit` (append-only), `users` (local with password hash, or `ad`), `sessions` (id = JWT `jti`, expires_at, revoked_at).

## Conventions
- Every mutating `signal.Service` method takes `actor` and writes one audit row (result `ok` or error text).
- Config unknown keys are refused; secrets are never echoed in errors.
- Timestamps stored UTC; display timezone from `timezone` config.
- Query helpers take `ctx` and `*sqlx.DB` as first params; `db.ErrNotFound` / `db.ErrDuplicate` are the only db sentinel errors.
- Web: Svelte 5 runes; `*.svelte.ts` hold shared state; each action dialog is a component mounted only while open, wrapping `Modal`.

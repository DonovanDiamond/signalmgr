# Code Blocks Reference (gateway)

`(s)` = method on `*Service` (or `*server` in `src/api`). Files only, no line numbers; find items with grep.

## cmd/
- `main`, `loadConfig` — cmd/gateway/main.go — wiring and graceful shutdown; a missing config file means all defaults
- `main` — cmd/fakesignal/main.go — dev server for the fake upstream

## src/config/config.go
- `EnvAdminUsername`, `EnvAdminPassword` — env overrides for the first admin
- `MinPasswordLength` — shortest accepted local password (12)
- `Config`, `JWTConfig`, `ADConfig`, `FirstAdminConfig` — typed `config.yaml`
- `Load` — read, default, env-override, validate (unknown keys refused)
- `(Config) Location` — configured `*time.Location`

## src/db/
- db.go: `ErrNotFound`, `ErrDuplicate`, `Open` — sentinels; open SQLite and run migrations
- migrate.go: `Migrate` — apply embedded `migrations/*.sql`, returns applied names
- numbers.go: `Number`, `ListNumbers`, `GetNumber`, `InsertNumber`, `UpsertNumberLabel`, `SetRegisterRequested` — labels and pending-registration marker
- audit.go: `AuditEntry`, `InsertAudit`, `ListAudit` — append-only audit log, newest first
- users.go: `SourceLocal`, `SourceAD`, `User`, `Session`, `CountUsers`, `GetUserByUsername`, `InsertLocalUser`, `UpsertADUser`, `InsertSession`, `GetSessionUser`, `RevokeSession`, `DeleteExpiredSessions` — users and sessions

## src/auth/
- auth.go: `RoleAdmin`, `RoleUser` — role names
- auth.go: `ErrInvalidCredentials`, `ErrDirectoryUnreachable`, `ErrInvalidToken`, `ErrNoRole` — sentinel errors
- auth.go: `User`, `Service`, `New` — public user view; auth service (seeds first admin, loads keys)
- auth.go: (s) `Login`, `Verify`, `Logout` — credentials to JWT; token to user (checks session row); revoke session
- keys.go: `KeyPair`, `LoadOrGenerateKeys` — JWT signing keys, generated on first run when both files absent
- ad.go: `adDirectory` (private), `Authenticate` — LDAP bind, group-to-role mapping via `group_map`

## src/signal/signal.go
- `StateRegistered`, `StateAwaitingCode`, `StateNotRegistered` — derived number states
- `ErrInvalid`, `ErrUnreachable` — input rejected; upstream down
- `Device`, `Number`, `GatewayInfo` — JSON views returned by the API
- `Service`, `New` — service over db + `signalmgr`
- `ValidateNumber` — E.164 check (other validators are private)
- (s) `Gateway`, `Numbers`, `Audit` — read: upstream about, merged number list, audit rows
- (s) `AddNumber`, `SetLabel` — db-only writes
- (s) `Register`, `Verify`, `Unregister` — registration lifecycle
- (s) `RemoveDevice`, `LinkQR`, `LinkURI` — linked devices
- (s) `SetPin`, `RemovePin`, `RateLimit` — PIN and rate-limit challenge

## src/fakesignal/fakesignal.go
- `ValidCode`, `RateLimitedNumber` — magic verification code; number that triggers a rate-limit challenge
- `Server`, `New(seed)`, (Server) `ServeHTTP` — in-memory upstream; `seed` preloads numbers
- private handlers (about, accounts, register, verify, unregister, devices, qrcode, link URI, pin, rate-limit) mirror the upstream paths

## src/api/
- server.go: `New` — builds the Fiber app, routes, limiter, SPA fallback
- server.go: `requireAuth`, `requireAdmin`, `currentUser` — auth middleware and context user
- server.go: `fail`, `errorHandler` — map `auth`/`signal` errors to statuses and JSON `{error}`
- handlers.go: `login`, `logout`, `me`, `gateway`, `numbers`, `addNumber`, `setLabel`, `register`, `verify`, `unregister`, `removeDevice`, `setPin`, `removePin`, `rateLimit`, `linkQR`, `linkURI`, `audit` — one per route
- handlers.go: `body[T]`, `done`, `finish` — decode body, 204, error-or-204

## web/
- embed.go: `FS` — embedded `build/` as `fs.FS`; error if not built

## web/src/lib/ (TypeScript)
- api.ts: `ApiError`, `apiFetch`, `apiBlob`, `seg` — fetch with bearer token, blob fetch, path-segment escape
- act.ts: `act` — run a mutation, toast success/error, return ok boolean
- session.svelte.ts: `session` — token/user store with `login`, `clear`, `logout`, `isAdmin`; persisted
- theme.svelte.ts: `theme` — light/dark store with `current`, `toggle`; persisted
- toast.svelte.ts: `toasts`, `dismiss`, `toastError`, `toastOk` — toast queue
- format.ts: `formatNumber`, `E164`, `RECENT_MS`, `ageMs`, `ago` — number and age formatting, validation regex
- types.ts: `Role`, `User`, `NumberState`, `Device`, `SignalNumber`, `GatewayInfo` — API shapes

## web/src/routes/
- `+layout.ts`, `+layout.svelte` — SPA mode; session guard (only /login without a session)
- `+page.svelte` — number list and dialog orchestration
- `login/+page.svelte` — sign-in form

## web/src/lib/components/
- `Modal` — shared `<dialog>` shell (title, testid, onclose)
- `Topbar`, `ThemeToggle`, `Toasts` — header, theme switch, toast popovers
- `NumberRow` — one number: state, label, devices, actions
- `AddNumberDialog`, `RegisterDialog`, `VerifyDialog`, `UnregisterDialog` — add / register / verify / unregister flows
- `LinkDeviceDialog` — QR/URI link and wait for the new device
- `RemoveDeviceDialog`, `PinDialog`, `RateLimitDialog` — remove device; set/remove PIN; rate-limit challenge

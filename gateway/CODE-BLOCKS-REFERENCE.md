# Code Blocks Reference (gateway)

`(s)` = method on `*Service` (or `*server` in `src/api`). Files only, no line numbers; find items with grep.

## cmd/
- `main`, `loadConfig` — cmd/gateway/main.go — wiring and graceful shutdown; a missing config file means all defaults
- `main` — cmd/fakesignal/main.go — dev server for the fake upstream (`-seed design`, `-link-after`)
- `main` — cmd/devseed/main.go — writes the design's labels into an empty gateway db (`-db`); does nothing if `numbers` has rows

## src/config/config.go
- `EnvAdminUsername`, `EnvAdminPassword` — env overrides for the first admin
- `MinPasswordLength` — shortest accepted local password (12)
- `Config`, `JWTConfig`, `ADConfig`, `FirstAdminConfig` — typed `config.yaml` (`trusted_proxies`, `jwt.access_ttl`/`refresh_ttl`, `ad.timeout`)
- `Defaults` — the configuration used when no file exists
- `Load` — read file over `Defaults`, then `Resolve` (unknown keys refused)
- (Config) `Resolve` — apply env overrides and validate; a caller using `Defaults()` must call it
- `(Config) Location` — configured `*time.Location`

## src/db/
- db.go: `ErrNotFound`, `ErrDuplicate`, `Open` — sentinels; open SQLite and run migrations
- migrate.go: `Migrate` — apply embedded `migrations/*.sql`, returns applied names
- numbers.go: `Number`, `ListNumbers`, `GetNumber`, `InsertNumber`, `UpsertNumberLabel`, `SetRegisterRequested` — labels and pending-registration marker
- audit.go: `AuditEntry`, `InsertAudit`, `ListAudit` — append-only audit log, newest first
- users.go: `SourceLocal`, `SourceAD`, `User`, `Session`, `CountUsers`, `GetUserByUsername`, `GetUserByID`, `ListUsers`, `InsertLocalUser`, `UpsertADUser`, `SetUserDisabled`, `SetUserPassword`, `InsertSession`, `GetSessionUser`, `ClaimRefresh`, `RevokeSession`, `RevokeUserSessions`, `DeleteExpiredSessions` — users and sessions; `ClaimRefresh` revokes and returns the user of a live refresh token in one statement

## src/auth/
- auth.go: `RoleAdmin`, `RoleUser` — role names
- auth.go: `ErrInvalidCredentials`, `ErrDirectoryUnreachable`, `ErrInvalidToken`, `ErrNoRole`, `ErrDisabled`, `ErrInvalidInput` — sentinel errors
- auth.go: `DirectoryTimeoutError` — `ErrDirectoryUnreachable` after `ad.timeout`
- auth.go: `User`, `Tokens`, `Service`, `New` — public user view; access + refresh token pair; auth service (seeds first admin, loads keys)
- auth.go: (s) `Login`, `Refresh`, `Verify`, `Logout` — credentials to tokens; spend a refresh token for a new pair and session; token to user (checks session row); revoke session
- users.go: `UserInfo` — admin's view of an account (adds `disabled`)
- users.go: (s) `ListUsers`, `CreateUser`, `SetDisabled`, `SetPassword`, `ChangePassword` — user management; every one is audited; `SetPassword` and disabling revoke the user's sessions, `ChangePassword` does not
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
- (s) `LinkDevice` — add a device to a number from a `sgnl://linkdevice?` URI; audited without the URI
- (s) `SetPin`, `RemovePin`, `RateLimit` — PIN and rate-limit challenge

## src/fakesignal/fakesignal.go
- `ValidCode`, `RateLimitedNumber` — magic verification code; number that triggers a rate-limit challenge
- `Server`, `New(seed)`, (Server) `ServeHTTP` — in-memory upstream; `seed` preloads numbers
- `Server.LinkAfter` — delay after a QR/link-URI request before a new registered account appears (0 = never)
- (Server) `SeedDesign` — adds the numbers shown in the UI design mockups
- private handlers (about, accounts, register, verify, unregister, devices list/add/remove, qrcode, link URI, pin, rate-limit) mirror the upstream paths

## src/api/
- server.go: `New` — builds the Fiber app, routes, per-route limiters, trusted proxies, SPA fallback
- server.go: `requireAuth`, `requireAdmin`, `currentUser` — auth middleware and context user
- server.go: `fail`, `errorHandler` — map `auth`/`signal` errors to statuses and JSON `{error}`
- handlers.go: `session` — body of a sign-in or refresh
- handlers.go: `login`, `refresh`, `logout`, `me`, `changePassword`, `users`, `addUser`, `patchUser`, `resetPassword`, `gateway`, `numbers`, `addNumber`, `setLabel`, `register`, `verify`, `unregister`, `removeDevice`, `linkDevice`, `setPin`, `removePin`, `rateLimit`, `linkQR`, `linkURI`, `audit` — one per route
- handlers.go: `body[T]`, `done`, `finish` — decode body, 204, error-or-204

## web/
- embed.go: `FS` — embedded `build/` as `fs.FS`; if not built, the `notbuilt/` placeholder page plus an error
- build/.gitkeep — keeps the embed directory present in a clean checkout

## web/src/lib/ (TypeScript)
- api.ts: `ApiError`, `apiFetch`, `apiBlob`, `seg` — fetch with bearer token, blob fetch, path-segment escape
- act.ts: `act` — run a mutation, toast success/error, return ok boolean
- session.svelte.ts: `session` — access/refresh token and user store with `login`, `refresh`, `clear`, `logout`, `isAdmin`; persisted in sessionStorage
- theme.svelte.ts: `theme` — light/dark store with `current`, `toggle`; persisted
- toast.svelte.ts: `toasts`, `dismiss`, `toastError`, `toastOk` — toast queue
- format.ts: `formatNumber`, `E164`, `RECENT_MS`, `ageMs`, `ago` — number and age formatting, validation regex
- types.ts: `Role`, `User`, `NumberState`, `Device`, `SignalNumber`, `GatewayInfo`, `ManagedUser` — API shapes

## web/src/routes/
- `+layout.ts`, `+layout.svelte` — SPA mode; session guard (only /login without a session)
- `+page.svelte` — number list and dialog orchestration
- `users/+page.svelte` — admin user list: create, disable/enable, reset password
- `login/+page.svelte` — sign-in form

## web/src/lib/components/
- `Modal` — shared `<dialog>` shell (title, testid, onclose)
- `Topbar`, `ThemeToggle`, `Toasts` — header, theme switch, toast popovers
- `AccountMenu`, `ChangePasswordDialog` — signed-in menu (Users link for admins, change own password, sign out); change-password form
- `CreateUserDialog`, `ResetPasswordDialog` — admin: create a local user; set a new password
- `NumberRow` — one number: state, label, devices, actions
- `AddNumberDialog`, `RegisterDialog`, `VerifyDialog`, `UnregisterDialog` — add / register / verify / unregister flows
- `LinkGatewayDialog` — QR/URI so a phone's account links to this gateway; polls until a new account number appears
- `AddDeviceDialog` — paste a `sgnl://linkdevice?` URI to add a device to one number (`POST /numbers/:number/devices`)
- `RenameDialog` — set a number's label (also used right after a gateway link)
- `RemoveDeviceDialog`, `PinDialog`, `RateLimitDialog` — remove device; set/remove PIN; rate-limit challenge

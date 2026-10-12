# Questions and decisions

See also [README.md](README.md) and [IMPLEMENTATION.md](IMPLEMENTATION.md). Each entry is marked **Decided** (made, recorded for the next person) or **Open** (needs the owner's review). Decisions 1 to 8 were made by the Governor while the owner was away; the rest were reported by the other coders.

## 1. Where does the gateway live?

**Decided.** A nested Go module at `gateway/` (`github.com/DonovanDiamond/signalmgr/gateway`). This keeps fiber, sqlx and the other dependencies out of the library's `go.mod`.

## 2. Which browsers?

**Decided.** Latest evergreen browsers only. This is the owner's override of the house rule that Firefox ESR 115 must render the UI; no PostCSS ESR targets are set up.

## 3. How do people sign in?

**Decided.** Local accounts plus optional AD (off unless configured), JWTs signed with a generated key pair, roles `admin` and `user`. The first run creates an administrator from config or the environment, or logs a generated one-time password.

## 4. Who may change things?

**Decided.** `user` may view everything; only `admin` may add, register, verify and unregister numbers, link and remove devices, set and remove PINs, and submit rate-limit challenges. Reading the audit log is admin-only as well. Link QR and URI are admin-only because they are part of linking a device.

## 5. Where do labels and "awaiting code" live?

**Decided.** In the gateway's database, because signal-cli-rest-api has no such concepts. State is registered (in Signal's account list), awaiting_code (register called, not verified, recorded locally) or not_registered (recorded locally only).

## 6. Audit log

**Decided.** Every mutating action is recorded (who, what, number, when, result) in an append-only table. Rows are never deleted. Link calls are the exception (see below).

## 7. How is the captcha obtained?

**Decided.** The person solves it at <https://signalcaptchas.org/registration/generate.html> and pastes the `signalcaptcha://...` token. Rate-limit challenges use <https://signalcaptchas.org/challenge/generate.html>.

## 8. How is it developed and tested without a Signal server?

**Decided.** Against an in-memory fake of signal-cli-rest-api written in Go (`src/fakesignal`). No real server and no usable Docker were available. Behaviour against a real server is unverified.

## Numbers are keyed by E.164 text

**Decided.** The number string, validated as `+` then 7 to 15 digits, is the primary key and the URL parameter. No surrogate id.

## Does a local account ever fall back to AD?

**Decided: no.** A username with a local account is judged by its local password only, so an AD outage or a same-named AD account cannot affect or take over a local account.

## How are AD users stored?

**Decided.** As rows with source `ad` and no password hash, refreshed (name and role) on each sign-in.

## First administrator password

**Decided.** When none is configured, a random password is generated and logged once at WARN. There is no other channel on a first run. Anyone who can read the log can read it, so it should be changed; see the next entry.

## Password length and changing passwords

**Decided.** Local passwords are 12 to 72 bytes (72 is bcrypt's input limit). Any local user changes their own password with `POST /api/auth/password` (current password required); an administrator can reset another local user's. AD accounts change theirs in the directory.

## Refresh tokens

**Decided.** Sign-in returns a 15-minute access JWT and an opaque refresh token valid for 8 hours (`jwt.access_ttl`, `jwt.refresh_ttl`), as the house style asks. Only the refresh token's SHA-256 is stored, in the `sessions` row. Sign-out revokes the session.

## Link naming

**Decided.** Two different things, named apart in the UI. "Link this gateway to a phone" (`LinkGatewayDialog`) shows a QR for a phone's Signal account to adopt the gateway as a linked device; it succeeds when a new account number appears in the list, then offers to label it. "Add a device" (`AddDeviceDialog`) takes a `sgnl://linkdevice?` URI from a new device and adds that device to one existing number.

## Refresh rotation

**Decided.** Each refresh spends the token and makes a new session with a new refresh token, and the old session is revoked. There is no reuse detection: a replayed spent token is simply refused (401), and does not revoke the new session. A refresh carries the original refresh expiry forward unchanged, so `refresh_ttl` is absolute, counted from sign-in.

## Input limits for users

**Decided.** Usernames match `[A-Za-z0-9._@-]{1,64}`, display names are 1 to 100 characters, passwords 12 to 72 bytes.

## Which password changes end sessions

**Decided.** Changing your own password does not revoke your other sessions. An administrator's reset of a password, and disabling a user, revoke all of that user's sessions.

## AD users cannot be disabled

**Decided.** Disabling is for local accounts; an AD account is disabled in the directory (the admin API refuses with 400). A disabled local user who gives the correct password gets 403 saying the account is disabled, and a wrong password still gets 401. A disabled user's refresh token is refused (401).

## Trusted proxies

**Decided.** `trusted_proxies` takes CIDRs only (`/32` for a single host); bare addresses are refused. Only requests from those peers may set the client IP, through `X-Forwarded-For`, which fiber reads right to left and validates as an IP. With none configured the header is ignored, since any client could forge it.

## Refresh token storage in the browser

**Decided.** Access and refresh tokens are kept in `sessionStorage`, so they last for the tab and are gone when it closes. Never `localStorage`.

## AD timeout leaves a goroutine behind

**Open.** `ad.timeout` stops the sign-in waiting, but `go-ad-auth` has no deadline, so the LDAP call is abandoned on its own goroutine until the OS gives up on the connection. Repeated timeouts against a dead directory can accumulate goroutines. The failed-sign-in limit slows this but does not prevent it.

## `jwt.token_ttl` is now rejected

**Open.** The key was replaced by `access_ttl` and `refresh_ttl`, and unknown keys are refused, so an existing `config.yaml` that sets `token_ttl` stops the gateway starting with a parse error. Breaking for existing installations; the alternative is to accept and ignore it.

## bcrypt cost

**Decided.** Cost 10 (bcrypt's default).

## AD bind has no timeout

**Decided.** `ad.timeout` (default 10s) bounds how long a sign-in waits and a timeout is reported as 503 with the timeout in the message. The library still has no deadline; see "AD timeout leaves a goroutine behind". Local accounts are unaffected.

## PostgreSQL

**Decided: not supported.** SQLite only. The house default allows PostgreSQL as optional; nothing here needs it.

## One Signal service per process

**Decided.** `signalmgr.API_URL` is a package global in the library, so a process can talk to exactly one signal-cli-rest-api server.

## Device 1 cannot be removed

**Decided.** `RemoveDevice` rejects device id 1 (the primary) with a 400; removing it would unregister the account, which has its own explicit action.

## Link calls are not audited

**Decided.** Generating a link QR or URI changes nothing until the phone scans it, and the UI polls and refreshes it. The device that appears afterwards is visible in the number's device list.

## Captcha format

**Decided.** The captcha must start with `signalcaptcha://`; anything else is rejected with a message saying so rather than being repaired.

## Login rate limit

**Decided.** 10 failed sign-ins per minute per client IP, using fiber's in-memory limiter; successful sign-ins are not counted. Behind a reverse proxy all clients share the proxy's address unless `trusted_proxies` is set (see "Trusted proxies"). `POST /auth/refresh` has its own counter of the same size.

## Successful mutations return 204

**Decided.** No body, matching the UI client, which handles 204. Creating a number does not return 201 with the row; the UI reloads the list.

## Missing config file

**Decided.** `-config` pointing at a missing file means all defaults. `main` uses `config.Defaults().Resolve()`.

## Shared version, go.work and release

**Decided.** The library and the gateway carry one version: tag `vN` (library) and `gateway/vN` (gateway), with git tags as the only source of truth. `gateway/go.mod` requires the library at that same version and has no `replace` (the earlier `replace ../` made the gateway module uninstallable). Local development uses a committed `go.work` at the repository root (`use . ./gateway`). `make release VERSION=vN` tags the library first (its module zip excludes `gateway/`, so its hash is unaffected), then pins and tags the gateway. The image tag is the `git describe` of the `gateway/v*` tags with the prefix stripped; `:latest` only for exact releases.

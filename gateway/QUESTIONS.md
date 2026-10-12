# Questions and decisions

See also [README.md](README.md) and [IMPLEMENTATION.md](IMPLEMENTATION.md). Each entry is marked **Decided** (made, recorded for the next person) or **Open** (needs the owner's review). Decisions 1 to 8 were made by the Governor while the owner was away; the rest were reported by the other coders.

## 1. Where does the gateway live?

**Decided.** A nested Go module at `gateway/` (`github.com/DonovanDiamond/signalmgr/gateway`) with `replace github.com/DonovanDiamond/signalmgr => ../`. This keeps fiber, sqlx and the other dependencies out of the library's `go.mod`.

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

**Open.** Minimum local password length is 12. There is no change-password endpoint yet, so a generated first-admin password cannot be changed from the UI.

## No refresh token

**Open.** The session is one 8-hour JWT backed by a revocable row in `sessions`. The house style asks for short-lived access tokens with refresh tokens; this deviates to avoid a refresh flow in a small tool. Sign-out revokes the session.

## bcrypt cost

**Decided.** Cost 10 (bcrypt's default).

## AD bind has no timeout

**Open.** `go-ad-auth` takes no context, so a dead directory server can hold a sign-in request until the OS gives up. Local accounts are unaffected.

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

**Decided.** 10 failed sign-ins per minute per client IP, using fiber's in-memory limiter; successful sign-ins are not counted. Behind a reverse proxy all clients share the proxy's address unless fiber is configured to trust proxy headers, which it is not.

## Successful mutations return 204

**Decided.** No body, matching the UI client, which handles 204. Creating a number does not return 201 with the row; the UI reloads the list.

## Missing config file

**Decided.** `-config` pointing at a missing file means all defaults. `config.Load` has no exported defaults, so `main` loads an empty temporary file. If `config` grows an exported `Defaults()`, use it instead.

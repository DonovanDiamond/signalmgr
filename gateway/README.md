# Signal Gateway

See also [IMPLEMENTATION.md](IMPLEMENTATION.md) (how it is built) and [QUESTIONS.md](QUESTIONS.md) (why it is built that way, and what is still open).

Signal Gateway is a small web application for registering and managing Signal phone numbers through a [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) server. It ships as one executable that serves both the web UI and its JSON API. The installation name shown in the UI is configurable.

## What it does

A **number** is a phone number in E.164 form (a leading `+`, then 7 to 15 digits, the first not zero), for example `+27821234567`. Each number has a state:

- **registered**: signal-cli-rest-api knows the account. Its linked devices are listed; device 1 is the primary.
- **awaiting_code**: registration was requested here and no verification code has been entered yet.
- **not_registered**: the number was added here and nothing has been requested.

Labels such as "Reception" and the awaiting-code state exist only in the gateway's own database. Numbers that exist in Signal but were never added here appear unlabelled.

An administrator can add a number, change its label, register it (a captcha token is required, by SMS or voice), verify it with the 6-digit code and an optional PIN, unregister it, link a new account to this gateway by QR code, add a device to a number by pasting a `sgnl://linkdevice?` link, remove a linked device, set or remove the registration PIN, and submit a rate-limit challenge.

The captcha is solved by the person at <https://signalcaptchas.org/registration/generate.html>. They paste the resulting token, which must start with `signalcaptcha://`. Rate-limit challenges use <https://signalcaptchas.org/challenge/generate.html>.

Example: registering `+27821234567` with the token `abc` is rejected with `captcha must be the token starting with signalcaptcha://`. The gateway never repairs a token.

Every change is written to an audit log (who, what, which number, when, result). Audit rows are never deleted. Verification codes, PINs and captcha tokens are never logged or stored.

## Who may do what

There are two roles. `user` may look at everything except the audit log. `admin` may do everything, including reading the audit log, every action above, and managing local users (create, disable or enable, reset a password). Any local user may change their own password. A role is enforced on each route, not in the page.

Sign-in is by local account, or by Active Directory when configured. With AD the role comes from group membership and cannot be edited here; someone in none of the mapped groups cannot sign in. A local account is judged by its own password only, so an AD outage never locks out the administrator. If AD cannot be reached, sign-in says exactly that rather than reporting a wrong password.

On an empty database the first administrator is created from config or the environment (`SIGNALGW_ADMIN_USERNAME`, `SIGNALGW_ADMIN_PASSWORD`). With no password configured, one is generated and written to the log once, at WARN. Local passwords must be 12 to 72 bytes. A disabled user cannot sign in even with the right password (403); Active Directory users cannot be disabled here.

Sign-in gives a 15-minute access token and a refresh token that the browser spends automatically for a new pair, so a session ends 8 hours after sign-in, however active (`jwt.access_ttl`, `jwt.refresh_ttl`), or when signed out. Failed sign-ins and failed refreshes are each limited to 10 per minute per client address. An administrator's password reset or disabling a user ends that user's sessions; changing your own password does not end your other sessions.

## Running it

```
make web          # builds the UI into web/build
go run ./cmd/fakesignal      # a fake signal-cli-rest-api on 127.0.0.1:8081, for trying it out
go run ./cmd/gateway -config config.yaml
```

With no `config.yaml` every setting takes its default: listen on `:8080`, database `data/gateway.db`, Signal API at `http://127.0.0.1:8081`. See [config.example.yaml](config.example.yaml). TLS is left to a reverse proxy. Behind one, list its address in `trusted_proxies` (CIDRs, `/32` for a single host) so the limit sees the real client; otherwise all clients share the proxy's address. `ad.timeout` (default 10s) bounds how long a sign-in waits for the directory. `jwt.token_ttl` from earlier versions is no longer accepted: replace it with `access_ttl` and `refresh_ttl`.

`make build-linux` and `make build-windows` produce static binaries in `dist/` (`CGO_ENABLED=0`). The UI is embedded when the binary is compiled, so run `make web` first (the build targets do).

### Previewing the UI with sample data

```
make web
go run ./cmd/fakesignal -seed design      # sample numbers; a new account appears 8 s after a link QR is requested (-link-after 0 = never)
go run ./cmd/devseed -db data/gateway.db     # labels for them; only into an empty numbers table
go run ./cmd/gateway -config config.yaml
```

Run `devseed` against the same `db` the gateway uses, before or after the gateway has created it. Without `make web` the gateway serves a page saying the UI is not built.

Latest evergreen browsers are the target; Firefox ESR 115 is deliberately not (see QUESTIONS.md).

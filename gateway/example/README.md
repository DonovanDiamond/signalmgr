# Signal Gateway, Docker Compose example

A working system: the gateway plus a real [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) (`bbernhard/signal-cli-rest-api:0.101`, `MODE=json-rpc`). Run everything from this directory. The image is built from the repository root (`gateway/Dockerfile`), so the first start takes a few minutes.

## Start

```
make up
make logs
```

`make` with no target lists every target. The targets wrap `docker compose -f compose.yaml ...`; `make up` is `docker compose up -d --build`. They also work from elsewhere with `make -C gateway/example up`.

The targets run docker through `sudo` by default; pass `DOCKER=docker` (for example `make up DOCKER=docker`) to skip it.

The gateway is at <http://127.0.0.1:8090>. It is published on the loopback address only; it speaks plain HTTP, so put a reverse proxy that terminates TLS in front of it before anyone else uses it, and list the proxy in `trusted_proxies` in `config.yaml`.

## Sign in

The first administrator is `admin`. On the first start with an empty database a password is generated and logged once:

```
make password
```

To choose the password instead, copy `.env.example` to `.env` and set `SIGNALGW_ADMIN_PASSWORD` (12 to 72 characters) before the first start. It is read only while the database is empty; to change it later, use the Users page or reset everything (below). Change a generated password after signing in.

## Register a real number

1. Open the Numbers page and add the number in E.164 form, for example `+27821234567`.
2. Choose Register. The gateway needs a captcha token: open <https://signalcaptchas.org/registration/generate.html>, solve it, and copy the link it offers (it starts with `signalcaptcha://`). Paste it whole.
3. Choose SMS or voice. Enter the 6-digit code you receive, and the PIN if the number has one.

The account lives in the `signal-data` volume, not in the gateway.

## Link to an existing phone

Use the link action on the Numbers page. A QR code appears; on the phone open Signal, Settings, Linked devices, add a device, and scan it. The new account shows up in the list once the phone confirms.

## Try it without a phone number

```
make fake
```

This adds `fakesignal` (an in-memory stand-in for signal-cli-rest-api with sample numbers), a one-shot `fake-seed` that labels them, and a second gateway, `gateway-fake`, at <http://127.0.0.1:8091> with its own database and its own first-admin password (`docker compose logs gateway-fake | grep password`, also printed by `make fake`). The real services keep running beside it; stop them with `docker compose stop gateway signal-api` if you only want the fake. Requesting a link QR makes a new fake account appear after 8 seconds. Nothing here talks to Signal.

## Reset everything

```
make reset
```

It asks you to type `yes`; `make reset FORCE=1` skips the prompt. This deletes all three volumes: the gateway databases and signing keys, and the registered Signal accounts in `signal-data`. Registrations are lost; a number may need re-registering. To wipe only the fake gateway, run `make fake-reset`. To stop without deleting anything, run `make down`. To reset only the real gateway, run `make down` and then `docker volume rm signal-gateway-example_gateway-data`.

## Files

- `compose.yaml`: services and volumes.
- `config.yaml`: gateway settings (paths are inside the container, under `/data`).
- `config.fake.yaml`: the same, pointed at `fakesignal`.
- `.env.example`: optional first-admin password.

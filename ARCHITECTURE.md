# signalmgr — Architecture

**What:** Go client library for [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api). Not a service; no `main`.
**Module:** `github.com/DonovanDiamond/signalmgr` — Go 1.23.2
**Target API:** signal-cli-rest-api 0.101
**Deps:** `gofiber/fiber/v2` (HTTP client agent), `gorilla/websocket` (receive in json-rpc mode)

## Layout
| Path | Role |
|---|---|
| `api.go` | `API_URL` (default `http://127.0.0.1:8080`) + private generic HTTP helpers `get/post/put/delete[T]` |
| `general.go` | Package-level calls not tied to an account: about, health, config, QR link, send, attachments |
| `account.go` | `Account` type; all per-account calls are methods on `*Account` |
| `signaltypes/types.go` | Payload structs for received messages (`MessageEnvelope`, `DataMessage`, `SyncMessage`, …). Types only. |

## Flow
caller → exported func / `*Account` method → `get/post/put/delete[T]` → `request` (`api.go`) → REST endpoint at `API_URL` → JSON decoded into `T`; API errors come back as `error`.

Receiving messages:
- normal/native mode: `(*Account).GetMessages` — REST poll
- json-rpc mode: `(*Account).GetMessagesSocket` — websocket, pushes `MessageResponse` into a channel

## Conventions
- Method names mirror HTTP verb + resource (`PostCreateGroup`, `DeleteGroupMembers`).
- Request bodies are often anonymous structs in the method signature.
- Per-account types are prefixed `Account_` (e.g. `Account_Configuration`).
- Optional query params: one `<Method>Options` struct per call (e.g. `GetGroupsOptions`); zero value = omitted. Built with `url.Values` and appended via `withQuery`.
- Binary responses (avatars, QR code, attachments) use `getRaw` and return `[]byte`.

## gateway/ (Signal Gateway app)
Separate Go module; see gateway/ARCHITECTURE.md and gateway/CODE-BLOCKS-REFERENCE.md.

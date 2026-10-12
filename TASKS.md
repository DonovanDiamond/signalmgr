## GOVERNOR (do not delete — re-read every time this file is opened)
- I am the Governor. I never write or edit code. Not one line.
- I route every task by lane (0 = me alone incl. git, 1 = one sonnet-coder, 2 = orchestrator). Lane 2 is the exception.
- Answer → haiku-scout. Change → sonnet-coder. Never a coder for a question.
- Verified / reported / missing are three different things. I never present the last two as the first.
- Nobody waits. Over budget = one check, then kill, then re-read git state before re-dispatching.
- Never re-run a suite that passed with no change since. Last tested: go test ./... passed on tree of v0.3.0
- Released: v0.2.0 at 9c9371d, v0.3.0, v0.4.0 (2026-10-12)
### Governor log
- 2026-10-12 — Committed T13 before reading 4 secret-scan hits; checked after: all benign. Rule: inspect scan hits before commit.
- 2026-10-12 — clean (session start, HEAD d65a5fb)
- 2026-10-12 — T2 manifest clean; brief changed mid-run (alias dropped), worker complied.
- 2026-10-12 — T17: first coder returned "placeholder" and left a server running; orchestrator recovered from diff and killed it.
- 2026-10-12 — T7 manifest accepted; signaltypes per-field claims unverified (structure-only check by orchestrator).

## Tasks
- [x] T1 — Generate ARCHITECTURE.md + CODE-BLOCKS-REFERENCE.md (Lane 0; survey via haiku-scout, declarations verified by grep). Committed 26dc7d2.
- [x] T2 — Renamed PostUnregistert → PostUnregister (old name dropped per user — BREAKING, bump version at next tag) + README names fixed. Build+vet rc=0 verified by Governor.
- [x] T3 — Added DeleteReaction to README, pushed (Lane 1, diff verified by Governor).
- [x] T4 — README signatures fixed (Post/DeleteReaction TargetAuthor, PostContact, PostCreateGroup); api.go errorResposne → errorResponse. Build+vet rc=0 verified by Governor.
- [x] T5 — PostGroupMembers/DeleteGroupMembers now send data (were nil). Diff + build/vet rc=0 verified by Governor; not tested against a live server.
- [x] T6 — README: return values on all 26 method bullets (reported count); 7 spot-checked vs code by Governor. Pushed.
- [x] T7 — Code aligned with signal-cli-rest-api 0.101 swagger (orchestrator, 4 coders, ~4 min reported). Governor verified: build/vet/gofmt rc=0, tag 0.101, contacts PUT, sync POST, /v1/search/{number}, health 204, config path, api.go read in full. signaltypes field list = reported only.
- [x] T8 — Exported APIError (status, message, account, challenge_tokens) replaces errorResponse. Diff read + build/vet rc=0 verified by Governor; non-2xx path not exercised. Pushed with T7.
- [x] T9 — 15 endpoints wrapped + receive/groups/contacts/qrcode options (orchestrator, 3 coders). Governor verified: build/vet/gofmt rc=0; polls, remote-delete, local-data, pin paths+methods and all 7 receive params match spec.
- [x] T10 — PathEscape on all path segments; discard-body calls use request(). Governor verified: no [any] left, no unescaped %s path, build/vet rc=0.
- [x] T11 — api_test.go added (3 tests). Found real bug: fasthttp decoded %2F; fixed via HostClient.DisablePathNormalizing. Governor verified: test FAILs without fix, all pass with it. Tagged v0.3.0.
- [ ] Backlog: tests cover api.go only; per-endpoint tests none.
- [x] T13 — Signal Gateway phase 1 committed 9659e5a. Governor verified: go test -count=1 all ok, vet/gofmt clean; LOOKED at 9 screenshots (ledger light/dark/390, login, register, verify, link, unregister, unreachable) — matches Exchange. Secret scan clean.
- [x] T14 — Phase 2 committed e4cd242. Governor verified: go test -count=1 all ok, vet/gofmt clean, staged-diff secret scan clean; LOOKED at 6 p2 shots (ledger light 1440, dark 390, add-device, users, account menu 390, verify). Reported only: proxy/refresh tests not run against old code.
- [x] T15 — Phase 3 polish (3 × Lane 1): no page shift/strip (daisyUI scroll lock off), account-menu header, 390 button widths, cross-platform .gitkeep, disabled buttons always explain why, muted example placeholders (4.77:1 / 5.24:1 reported). Governor LOOKED at p3/p4/p5 shots; go test -count=1 ok.
- [x] T16 — gateway/Dockerfile + gateway/example/ (signal-cli-rest-api 0.101 json-rpc + gateway; profile fake). Governor read Dockerfile + compose; compose config parses (reported); native build+run+login against fake (reported). NOT run in Docker (daemon permission denied).
- [ ] Open for user: gateway/QUESTIONS.md (token_ttl alias, AD abandoned-bind cap, refresh reuse detection, + Phase 1 items). Gateway pushed 2026-10-12 (origin/main = 53d4017).
- [x] T12 — Device-registration UI: 5 design renditions (published v3, all 5 files read by Governor before publish; not rendered/screenshotted) on a Design canvas (https://claude.ai/artifact/5KafbQcfDsVFEMazbRhsh5). Plan: scratchpad/plans/ui-renditions.md. 5 × sonnet-coder in parallel, one artboard each (disjoint). 2026-10-12, budget 6 min each. User override: evergreen browsers, not Firefox ESR 115 → record in QUESTIONS.md when the app is scaffolded.
- [x] T17 — Shared versions (vN + gateway/vN, gateway requires root vN, go.work, no replace) + root Makefile (image with git-describe tag, release). Lane 2 (build config + go.mod). Plan: scratchpad/plans/release-plan.md. 2026-10-12, budget 25 min. Governor verified: diff read (Makefile, go.work, Dockerfile, go.mod, main.go), gateway build+vet ok (GOFLAGS=-mod=readonly). Reported: tests ok, make -n outputs. Not run: docker build. User env GOFLAGS=-mod=mod breaks plain go cmds under go.work. Released v0.5.0 via make release.
- [x] T18 — Phone-numberless accounts (upstream PR #896, unmerged): library AccountInfo/GetAccounts both shapes + receive fields; gateway account id = number|ACI, UI, fakesignal objects format. Lane 2. Plan: scratchpad/plans/numberless-plan.md. Governor verified: account.go + migration 003 diff read, AdminDelete/RemoteDelete targetTimestamp confirmed vs upstream docs.go, root go test ok, gofmt clean, LOOKED at nl-/nl2- shots. Reported: gateway tests, svelte-check, build. Not run vs real #896 (unmerged). Open for user: 'Numbers' heading → 'accounts'? release v0.6.0 (GetAccounts breaking).

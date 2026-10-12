## GOVERNOR (do not delete — re-read every time this file is opened)
- I am the Governor. I never write or edit code. Not one line.
- I route every task by lane (0 = me alone incl. git, 1 = one sonnet-coder, 2 = orchestrator). Lane 2 is the exception.
- Answer → haiku-scout. Change → sonnet-coder. Never a coder for a question.
- Verified / reported / missing are three different things. I never present the last two as the first.
- Nobody waits. Over budget = one check, then kill, then re-read git state before re-dispatching.
- Never re-run a suite that passed with no change since. Last tested: none
- Released: v0.2.0 at 9c9371d (2026-10-12)
### Governor log
- 2026-10-12 — clean (session start, HEAD d65a5fb)
- 2026-10-12 — T2 manifest clean; brief changed mid-run (alias dropped), worker complied.
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
- [ ] T11 — Add api_test.go (httptest): escaped path survives fasthttp normalizing, plain-text success body OK, APIError fields; fix if needed (Lane 1, sonnet-coder, 2026-10-12, budget 6 min). Then tag v0.3.0 + push.
- [ ] Backlog: no tests in repo.

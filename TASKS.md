## GOVERNOR (do not delete — re-read every time this file is opened)
- I am the Governor. I never write or edit code. Not one line.
- I route every task by lane (0 = me alone incl. git, 1 = one sonnet-coder, 2 = orchestrator). Lane 2 is the exception.
- Answer → haiku-scout. Change → sonnet-coder. Never a coder for a question.
- Verified / reported / missing are three different things. I never present the last two as the first.
- Nobody waits. Over budget = one check, then kill, then re-read git state before re-dispatching.
- Never re-run a suite that passed with no change since. Last tested: none
### Governor log
- 2026-10-12 — clean (session start, HEAD d65a5fb)
- 2026-10-12 — T2 manifest clean; brief changed mid-run (alias dropped), worker complied.

## Tasks
- [x] T1 — Generate ARCHITECTURE.md + CODE-BLOCKS-REFERENCE.md (Lane 0; survey via haiku-scout, declarations verified by grep). Committed 26dc7d2.
- [x] T2 — Renamed PostUnregistert → PostUnregister (old name dropped per user — BREAKING, bump version at next tag) + README names fixed. Build+vet rc=0 verified by Governor.
- [x] T3 — Added DeleteReaction to README, pushed (Lane 1, diff verified by Governor).
- [x] T4 — README signatures fixed (Post/DeleteReaction TargetAuthor, PostContact, PostCreateGroup); api.go errorResposne → errorResponse. Build+vet rc=0 verified by Governor.
- [ ] T5 — Bug: PostGroupMembers/DeleteGroupMembers send nil instead of data (account.go:222–235). Verified by Governor. Lane 1, sonnet-coder, 2026-10-12, budget 6 min.

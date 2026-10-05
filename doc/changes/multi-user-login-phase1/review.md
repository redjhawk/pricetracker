# Review: multi-user-login phase 1

Reviewed specification revisions: `doc/specifications/multi-user-login/{functional,delivery-plan,technical}.md`, `API_SPECIFICATION.md` at the revision below
Reviewed code revision: `3f90121684f4734c1cd9f2e0b7f24ed8ba0724f6` (diff `origin/master...HEAD`)
Reviewer: independent reviewer agent
Adjudicator: (pending, separate agent)

Verification run by the reviewer at HEAD: `go vet ./...` clean, `go test ./...` pass, `npm run build` pass. Per-commit builds could not be run (worktree/checkout commands were not permitted in this session); intermediate-commit findings below come from reading each commit's diff.

Checked without findings: access middleware matrix (auth routes open; admin routes 401/403; admin refused 403 on item and settings routes; protected mode without session 401; open mode uses owner 0); owner scoping of list, get, add (per-owner uniqueness), delete, refresh one/all, purchase goal, AI review request and display; scheduler and background review use the item's `owner_id` for the LeBoncoin session and Claude token; first-user inheritance runs in one transaction on the single SQLite connection; items/settings table rebuild runs in a transaction with `foreign_keys=OFF` (single connection, so the pragma applies), keeps `purchase_goal`, recreates `items_next_check`, and runs `foreign_key_check` before commit, so `price_observations`, `collection_attempts`, `ai_reviews` and offer rows are not cascaded away; session tokens are 32 random bytes stored as SHA-256; cookie `HttpOnly`, `SameSite=Lax`, `Path=/`, 30-day `Max-Age`, `Secure` on TLS only (as specified); unknown usernames run a dummy hash and are counted by the lockout; admin reset ends admin sessions; scripts quote fixed paths only.

## Findings

### REV-001: Concurrent failed logins clear an active lockout

- Subject/requirement: FR-AUTH-005, D-11.
- Evidence: `internal/service/auth.go` `Login` checks the lock, then spends a full PBKDF2 hash, then calls `recordLoginFailure`. `recordLoginFailure` resets the counter whenever `lockedUntil` is non-zero, without checking that the lock has expired (`if !failures.lockedUntil.IsZero() { failures = loginFailures{} }`).
- Scenario: an attacker keeps 6+ wrong-password requests for one username in flight. The 5th failure sets the lock; any request that passed the lock check before that then fails and resets the entry to `count=1` with no lock. Because each attempt takes seconds on the device (REV-002), keeping requests in flight is easy, so guessing continues at about the hash rate and the 1-minute lock is never enforced.
- Expected: once locked, failures within the lock keep it; the reset applies only when `now` is after `lockedUntil`.
- Suggested action: reset only when `!now.Before(failures.lockedUntil)` and the lock was set; add a service test that records a failure while locked.
- Provisional severity: critical (security requirement D-11 can be bypassed).

### REV-002: PBKDF2 600 000 iterations is likely several seconds per login on ARMv6, with no limit on parallel hashing

- Subject/requirement: technical.md "On ARMv6 a login costs roughly a second"; unspecified load limit.
- Evidence: `passwordIterations = 600_000` with HMAC-SHA-256 is about 1.2 million SHA-256 block compressions (about 77 MB hashed). Go's generic SHA-256 on an ARM1176 (Pi Zero, no crypto/NEON) reaches roughly 10 to 20 MB/s, so one hash takes about 4 to 8 s, not 1 s. The first unknown-username login also computes the dummy hash (twice the cost). No measurement on the device is recorded in `implementation.md`.
- Impact: slow logins and slow user creation; unauthenticated parallel logins (lockout is per username, so rotating usernames is not limited) can saturate the single core and stall the scheduler and API for everyone.
- Suggested action: measure on the target device; choose an iteration count that meets the stated about-1-second target (the format stores the count, so it can change without migration), and/or allow only one or two concurrent password checks. Update technical.md with the measured value.
- Provisional severity: non-critical (availability/usability; does not leak data), but should be settled before release.

### REV-003: Intermediate stacked commits are unsafe if merged alone

- Subject: AGENTS.md stage 8 (each split PR must stand on its own).
- Evidence: `f322b2b` (API endpoints) turns on protected mode and moves owner-0 items to the first user, but item queries are only scoped in `c36012d` and settings in `d297acb`; the login UI only arrives in `c1fbc8d`.
- Scenario: with `f322b2b` merged alone, once the admin creates two users, each user sees, edits, refreshes and deletes the other's items and shares one Claude token and LeBoncoin session (FR cross-user isolation broken); with `f322b2b`..`c36012d` merged without `c1fbc8d`, creating a user through the API makes the web UI return 401 everywhere with no login page.
- Suggested action: either merge the stack only as a unit (state so in each PR description and in `commit-step.md`), or reorder so ownership scoping (`d297acb`, `c36012d`) lands before the endpoints that can create users, and gate user creation until the UI commit. Per-commit `go test` was not verified by the reviewer; the coordinator should run it on each commit.
- Provisional severity: critical for the stacked-PR workflow (a partial merge exposes one user's data to another); not a defect of the final tree.

### REV-004: Unbounded in-memory lockout map

- Subject: FR-AUTH-005 (technical design), unspecified limit.
- Evidence: `s.loginFailures` gains one entry per distinct failed username and entries are only deleted on a successful login of that username.
- Impact: an unauthenticated client sending random usernames grows memory on a 512 MB device until restart. Slowed by REV-002, so low practical risk.
- Suggested action: drop entries whose lock expired, or skip storing entries when `count` would reset (for example prune expired ones on each failure).
- Provisional severity: non-critical.

### REV-005: Focus does not move to the invalid field after a failed "Add user"

- Subject: accessibility (frontend skill), admin page.
- Evidence: `src/components/AdminPage.tsx` `submitAdd` calls `setSubmitting(false)` then immediately `usernameRef.current?.focus()` / `passwordRef.current?.focus()`. The input is still rendered `disabled` until React re-renders, so `focus()` does nothing and the keyboard/screen-reader user is left on the modal button with no announcement of the field error.
- Suggested action: focus in an effect after the error state renders (as `LoginPage` does for its error).
- Provisional severity: non-critical.

### REV-006 (question): `admin-password` from the deploy wrapper prints the password over `ssh -t`

- Subject: deployment (user instruction), `scripts/deploy-armv6.sh`.
- Evidence: the password is printed to the SSH terminal, so it also lands in any local terminal scrollback or CI log that runs the wrapper. This is the documented design (DEPLOYMENT.md), so no change is required; recorded so the adjudicator can confirm it is accepted.
- Provisional severity: non-critical / question.

## Summary

6 findings: 2 provisionally critical (REV-001 lockout bypass under concurrency; REV-003 unsafe intermediate stacked commits), 3 non-critical (REV-002 PBKDF2 cost on ARMv6 and no parallel limit, REV-004 unbounded lockout map, REV-005 focus after admin form error), 1 question (REV-006). No auth bypass, admin data access or cross-user access was found in the final tree. Open-mode compatibility and migration data safety look correct. All findings go to the independent adjudicator.

## Re-review of fixes

Reviewed code revision: `6f122ebce878c8e6de82183c08ec8c487919c807` (diff from `3f90121` plus the reordered history). Reviewer ran `go vet ./...` (clean) and `go test ./...` (pass) at the tip. The per-part checks are taken from `implementation.md` (developer evidence). The reviewer did not repeat them because checkout was blocked.

- REV-001: verified fixed. `recordLoginFailure` no longer resets an entry whose lock is still active. A failure during the lock keeps `lockedUntil` unchanged. An expired lock is pruned before the next failure is counted, so counting restarts at 1. The new test `TestFailureDuringLockKeepsItAndOldEntriesArePruned` covers both cases. The intermediate commit `411ed69` still has the old bug, but it ships in the same part as `be988dd`, so this is acceptable.
- REV-002: verified fixed in code. The iteration count is now 100 000, about 6 times cheaper, which matches my estimate of roughly 1 s on ARMv6. `hashSlot` allows one PBKDF2 computation at a time and gives up when the request context ends. Stored hashes keep their own count. The cost has still not been measured on a real device; a QA step should time a login there.
- REV-003: verified fixed by reading each commit's diff. New order: store (`064d106`), then service and CLI (`411ed69`, `be988dd`), then per-user settings (`109fa63`), then item scoping (`efceed5`), then HTTP endpoints (`e4bccef`), then UI (`9ea4e2c`, `9f693ca`). No commit before `e4bccef` exposes any route that creates users or sessions. Owner scoping of settings and items is in place before that, so no intermediate part exposes one user's data to another. The remaining state between part 5 and part 6 (a user created through the raw API locks the web UI with 401) is recorded and accepted in `implementation.md`. `commit-step.md` does not exist yet; the coordinator still owes it.
- REV-004: verified fixed. On each failure, entries with an expired lock and unlocked entries older than one hour (`failureMemory`) are deleted. Choosing one hour is reasonable. It does not weaken D-11, which only needs 5 consecutive failures, and with the one-hash limit the map holds at most about 3 600 entries. Side effect: failures spread out more than one hour apart never add up to a lock. D-11 does not say otherwise, so this is acceptable. The pruning loop is O(n) under `s.mu`, which is negligible at that size.
- REV-005: verified fixed. Focus now moves in an effect after `submitting` becomes false and the error has rendered.
- REV-006: rejected by the adjudicator with reasons; no recheck needed.

### REV-007 (new): a single global hash slot lets one client delay every login

- Evidence: `hashSlot` in `internal/service/auth.go` has capacity 1 and requests wait in an unbounded queue until their context ends. One unauthenticated client sending continuous logins with rotating usernames keeps the slot busy, so legitimate logins (and admin user creation) wait about 1 s for each queued request, or time out.
- Impact: availability only; nothing leaks and the scheduler and item APIs are not affected. This is the trade-off REV-002 accepted, now concentrated on login.
- Suggested action: optional. Refuse a login with 429/503 when the slot is not free within a short time instead of queueing, or accept and record the trade-off.
- Provisional severity: non-critical.

Re-review summary: REV-001 to REV-005 are fixed, and the reorder removes cross-user exposure from every intermediate part. One new non-critical finding (REV-007). Still open: `commit-step.md` and a timed login on the real ARMv6 device.

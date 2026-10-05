# Review decisions: multi-user-login phase 1 (issue #15)

Reviewed specification revisions: `doc/specifications/multi-user-login/{functional,delivery-plan,technical}.md`, `API_SPECIFICATION.md` at `3f90121`
Reviewed code revision: `3f90121684f4734c1cd9f2e0b7f24ed8ba0724f6` (diff `origin/master...HEAD`)
Reviewer: independent reviewer agent (`review.md`)
Adjudicator: independent review adjudicator agent (separate from developer and reviewer)

## Findings and decisions

### REV-001: Concurrent failed logins clear an active lockout

- Evidence: `internal/service/auth.go:208-209` resets the entry whenever `lockedUntil` is non-zero, with no check that `now` is past it. `Login` reads the lock at line 167 before the slow hash, so requests already past the check reset a fresh lock when they fail. Confirmed by reading the code.
- Impact and scenario: an unauthenticated attacker keeps several wrong-password requests in flight for one username. The lock is overwritten as soon as it is set, so password guessing runs at the hash rate.
- Criticality: critical. FR-AUTH-005 / D-11 is a specified security control, and anyone who can reach the login endpoint can defeat it with almost no effort.
- Disposition: fix.
- Reason: this is a defect against a specified requirement. The fix is small.
- Specification decision: not applicable.
- Resolution: pending. Required action (developer): reset the counter only when a lock exists and has expired (`!failures.lockedUntil.IsZero() && !now.Before(failures.lockedUntil)`). A failure recorded while the lock is still active keeps the lock and does not shorten it. Add a service test that records a failure while locked and checks that the account stays locked until `lockedUntil`, then accepts attempts again after expiry. Reviewer recheck required.
- Follow-up: none after verification.

### REV-002: PBKDF2 at 600 000 iterations costs several seconds on ARMv6, and parallel hashing is unlimited

- Evidence: `auth.go:33` `passwordIterations = 600_000`. `technical.md` targets about 1 s per login on ARMv6. The reviewer's estimate is 4 to 8 s on an ARM1176 with generic Go SHA-256, which is plausible (no hardware SHA, single core). No measurement on the device is recorded. Nothing limits concurrent hashes, and lockout is per username, so attempts that rotate usernames are not limited.
- Impact and scenario: logins and user creation take several seconds. A few parallel unauthenticated requests saturate the only core and stall the scheduler and the API for every user.
- Criticality: non-critical. This affects availability and usability, not confidentiality, and a self-hosted LAN device lowers the likelihood. It still contradicts the technical spec's stated cost, so it is fixed now rather than deferred.
- Disposition: fix.
- Reason and tradeoff: OWASP's 600k for PBKDF2-SHA256 assumes server hardware where it costs about 0.1-0.3 s. On a Pi Zero, the same work costs 4-8 s, which is unusable, and it adds a CPU-exhaustion vector. The stored hash format records the iteration count, so the cost can be raised later without a migration. The concurrency cap removes the saturation vector whatever the count. Decision:
  1. Set `passwordIterations = 100_000`. That is about 1 s on ARMv6 by the reviewer's throughput estimate, which matches the spec target. It is still far above brute-force-trivial for a LAN app whose online guessing is also rate-limited by the lockout. Existing hashes keep the count stored in them. Verification keeps reading the count from the hash and does not rehash.
  2. Allow at most **1** concurrent PBKDF2 computation process-wide. Use a buffered channel of size 1 used as a semaphore. Waiting must honour `ctx` cancellation. This covers login (including the dummy hash for unknown usernames), user creation, password reset and the admin-password command path.
  3. Update `technical.md` with the chosen count, the rationale above and the cap. If the coordinator or QA can time a login on the target device, record the measured value in `implementation.md`. Otherwise record that the figure is an estimate.
- Specification decision: not applicable (technical).
- Resolution: pending (developer, then reviewer recheck).
- Follow-up: if a device measurement shows a login well under 0.5 s, raise the count in a later change. This needs no migration.

### REV-003: Intermediate stacked commits are unsafe if merged alone

- Evidence: in the current order, `f322b2b` (endpoints, user creation, protected mode) comes before `d297acb` (per-user settings) and `c36012d` (item scoping). Shortstats: dad9afd 438, 6372b1c 408, f322b2b 312, d297acb 359, c36012d 113, c1fbc8d 427, 3f90121 71. Every commit is ≤500 lines, so each commit can become one stacked PR.
- Impact and scenario: if the PRs up to `f322b2b` are merged without the next ones, any users created can read, modify and delete each other's items and share one Claude token and LeBoncoin session. That is cross-user data exposure on `master`.
- Criticality: critical. Stacked PRs are merged one at a time, so `master` really does pass through each intermediate state. "Merge as a unit" depends on process discipline and is not accepted as the only safeguard.
- Disposition: fix (reorder commits; do not amend existing published commits beyond the reorder needed before first push).
- Reason: putting ownership scoping before any way of creating users removes the exposure. The remaining intermediate state is accepted: after the endpoints PR is merged but before the frontend PR, a user created through the raw API makes the web UI return 401 with no login page. That state is merely unusable, is reachable only by deliberately calling the admin API or running the admin-password command, exposes no data, and is resolved by the next PR in the stack. It must be stated in the PR descriptions and in `commit-step.md`.
- Specification decision: not applicable.
- Resolution: pending. Required action (developer and coordinator):
  1. Rebuild the branch in this order: `dad9afd` (store) → `6372b1c` (auth service and admin-password command) → `d297acb` (per-user settings) → `c36012d` (item scoping) → `f322b2b` (HTTP login/session/admin endpoints) → `c1fbc8d` (web UI) → `3f90121` (docs). This branch is not yet pushed, so rebuilding it from new commits is allowed. Never amend or force-push commits that have already been pushed.
  2. If `d297acb` or `c36012d` depends on code in `f322b2b`, move only that dependency into the scoping commit or into a small preceding commit. The dependency is most likely the request-context owner helper or middleware. Moving it must not add any route that creates users or sessions. While the scoping and settings commits stand alone, the open-mode owner 0 must keep working. Keep every commit at most 500 changed lines.
  3. Note that `6372b1c` already allows creating the administrator through the CLI. An administrator cannot access items or settings (403), so no item exposure exists before ordinary users can be created. Confirm that the scoping commits keep that true.
  4. On every rebuilt commit, run `go vet ./...` and `go test ./...` (plus `npm run build` from the UI commit onward). Record the results per commit in `commit-step.md`. Also record the accepted intermediate state from the Reason above and the "merge in order" note in each PR description.
- Follow-up: coordinator verifies the per-commit checks before opening PRs.

### REV-004: Unbounded in-memory lockout map

- Evidence: entries are added for each failed username and removed only on successful login.
- Impact and scenario: random-username floods grow memory on a 512 MB device. With the REV-002 one-hash cap, the rate is about 1 entry per second, roughly 86k small entries per day. That is slow but unbounded.
- Criticality: non-critical. Exhausting memory needs a sustained attack lasting days, and a restart clears it.
- Disposition: fix (small, same function as REV-001).
- Reason: the change is a few lines in `recordLoginFailure`, which REV-001 already touches.
- Specification decision: not applicable.
- Resolution: pending. Required action: when recording a failure, delete entries whose lock has expired, or whose last failure is older than the lockout counting window. A simple linear prune under the existing mutex is enough. Add a test that expired entries are removed.
- Follow-up: none.

### REV-005: Focus does not move to the invalid field after a failed "Add user"

- Evidence: `src/components/AdminPage.tsx` `submitAdd` calls `focus()` in the same tick as `setSubmitting(false)`, while the input is still `disabled`. The focus call therefore has no effect.
- Impact and scenario: keyboard and screen-reader administrators are not taken to the field error.
- Criticality: non-critical. This is an accessibility defect, has a workaround (tab to the field) and leaks no data.
- Disposition: fix.
- Reason: the frontend skill requires accessible error handling. `LoginPage` already shows the correct pattern, so the fix is low-risk.
- Specification decision: not applicable.
- Resolution: pending. Required action: move the focus into an effect that runs after the error state renders and the input is enabled, mirroring `LoginPage`.
- Follow-up: QA checks keyboard focus on the add-user error.

### REV-006 (question): `admin-password` via the deploy wrapper prints the password over `ssh -t`

- Evidence: this behaviour is documented in DEPLOYMENT.md. The password appears in the operator's terminal and in any log that captures the wrapper's output.
- Impact and scenario: the operator must see the generated password once to use it. The exposure is limited to the operator's own terminal or log.
- Criticality: non-critical.
- Disposition: reject (no code change).
- Reason: showing a one-time generated password to the operator who asked for it is the intended function. Hiding it would need another delivery channel, which is not specified. The user can change the password after the first login, and the admin-password command can rotate it. Remaining risk: CI logs or a shared scrollback could retain it. DEPLOYMENT.md should already advise against running the command in logged or CI contexts. If it does not, the developer adds one sentence (documentation only).
- Specification decision: not applicable.
- Resolution: rejected (code); doc sentence check pending (developer).
- Follow-up: none.

### REV-007: A single global hash slot lets one client delay every login

- Evidence: `hashSlot` (capacity 1) in `internal/service/auth.go` at `6f122eb`. Waiting requests queue without limit until their context ends (re-review in `review.md`).
- Impact and scenario: an unauthenticated client that floods logins with rotating usernames delays or times out legitimate logins and admin user creation. Existing sessions (30-day cookies), item APIs and the scheduler are unaffected, and nothing leaks.
- Criticality: non-critical. Only new logins are affected, attacks need LAN reach on a self-hosted device, and the effect stops when the flood stops.
- Disposition: defer.
- Reason: REV-002 deliberately chose this tradeoff, moving the CPU-starvation risk away from the whole device and onto login only. Refusing logins with 429/503 would add a new response to the `POST /api/auth/login` contract. That needs a contract update and UI handling, and it would also refuse legitimate users during the same flood, so availability barely improves. A queue that stays bounded only by request contexts is acceptable for phase 1. Remaining risk: during a flood, users who are not logged in cannot log in.
- Specification decision: not applicable.
- Resolution: deferred.
- Follow-up: technical specifier/developer, in a later phase. Options are a per-client rate limit, or fast rejection with an API contract change. Track as a phase-2 item in `delivery-plan.md`.

## Verification update (re-review at `6f122eb`)

- REV-001: fixed, verified by reviewer.
- REV-002: fixed (100 000 iterations, one-hash cap), verified by reviewer. A timed login on the ARMv6 device is still not recorded.
- REV-003: fixed by the commit reorder; reviewer verified no cross-user exposure in any intermediate commit. Per-commit checks must still be recorded in `commit-step.md`.
- REV-004, REV-005: fixed, verified by reviewer.
- REV-006: rejected (unchanged).
- REV-007: deferred (above).

## Release readiness

Superseded by the verification update above: no critical finding remains open. Before completion, `commit-step.md` must record the per-commit checks for the reordered stack. QA is still to run.

Original status: Blocking: REV-001 (critical, fix pending) and REV-003 (critical, commit reorder and per-commit checks pending). The other fixes (REV-002, REV-004, REV-005) must be done before release, but none of them blocks on its own. REV-006 is rejected with reasons. No functional question for the user is open. Completion is blocked until the developer fixes and the reviewer rechecks REV-001 through REV-005, and until the coordinator records the per-commit checks for the reordered stack in `commit-step.md`. QA has not yet run.

# Test cases: long whitespace-only secret inputs are untouched (issue #286)

Sources: functional specification [settings-secret-masking](../../specifications/settings-secret-masking/functional.md) FR-SECRET-014 (with FR-SECRET-004 and FR-SECRET-008), the clarified row FR-LBC-SET-006 of [LeBoncoin session settings](../../specifications/leboncoin-session-settings/functional.md), FR-CLT-SET-004 of [Claude token settings](../../specifications/claude-token-settings/functional.md), and use case [UC-SECRET-007](../../use-cases/settings-secret-masking/UC-SECRET-007-save-with-long-whitespace-only-input.md).

Conventions:

- **Secrets:** as in the #265 catalogue: T = `sk-ant-oat01-synthetic-token-wxyz` (saved Claude token), U = `sk-ant-oat01-synthetic-new-uvst` (new token), S = `synthetic-session-value-9f3a` (saved session), N = `synthetic-renewed-or-new-n0p1` (new session), Y = `synthetic-renewed-value-y7k2` (session stored by a renewal).
- **Long blank inputs:** `W9000` = 9000 spaces; `Wmix` = 9000 characters mixing spaces, tabs, `\n` and `\r\n`; `W8192` / `W8193` = exactly 8192 / 8193 spaces; `C1025` / `C2000` = 1025 / 2000 spaces. Every save request built with them stays under the save request size limit (32 768 bytes) unless stated otherwise.
- **Untouched** means: the stored secret keeps its value, `revision`, `updatedAt` and warnings (session `status`, `expiresAt`, `revokedAt`, `lastAttempt`; token `lastRejectedAt`), and no Claude verification call is made.
- **Go integration tests** (`internal/httpapi`, `internal/service`) use `newSettingsServer` / `newSessionService` (real SQLite, normal migrations) and the fake Claude transport or `fakeClaude`, which count verification calls, and send `PUT /api/v1/settings` with the `leboncoinSession` / `claudeToken` parts of `API_SPECIFICATION.md`. They extend the #265 tests in `internal/httpapi/secret_masking_test.go`; test names carry a prefix of this change (for example `TestLongWhitespace_TC002_…`) so they do not collide with the #265 `TC-` numbers.
- **E2E tests** (Playwright, `tests/leboncoin-session-settings.spec.ts` and `tests/claude-token-settings.spec.ts`) mock the API with `page.route` as the existing specs do and record every request body.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-SECRET-014 | TC-001, TC-002, TC-003, TC-004, TC-005 |
| FR-LBC-SET-006 | TC-001, TC-002, TC-005 |
| FR-SECRET-004 | TC-002, TC-003 |
| FR-SECRET-008 | TC-004 |
| FR-CLT-SET-004 | TC-003, TC-005 |

| Use case | Test cases |
| --- | --- |
| UC-SECRET-007 | TC-001, TC-002, TC-003, TC-004, TC-005 |

## Server

### TC-001 — A whitespace-only session input of any length is no change, never "too long"

- Covers: FR-SECRET-014, FR-LBC-SET-006; UC-SECRET-007
- Level: unit (Go, the function that turns the `leboncoinSession` part into a session change: `sessionChangeFor`, or wherever the technical specification places the rule)
- Given the session part `{value: v, revision: r}`
- When it is converted for v = `""`, `"   "`, `W8192`, `W8193`, `W9000`, `Wmix`
- Then each result is "no session change" with no error.
- Corner cases (the length rule still applies to inputs that are not whitespace only):
  - 8193 `v` characters: `INVALID_SESSION`, message "The pasted text is too long (maximum 8192 characters).";
  - `W9000` followed by `abc` (and `abc` followed by `W9000`): longer than 8192 and not whitespace only, so `INVALID_SESSION` too long, never trimmed into a replacement with `abc`;
  - `W100` + `datadome=N` + `W100` (under 8192): a replacement with N, as before;
  - a remove part `{remove: true, revision: r}`: still a removal.

### TC-002 — Saving a long whitespace-only session input keeps the session untouched

- Covers: FR-SECRET-014, FR-SECRET-004, FR-LBC-SET-006; UC-SECRET-007 (main flow)
- Level: integration (Go HTTP handlers, real SQLite, fake Claude)
- Given S is saved at revision r with a recorded `lastAttempt` (`rejected`), and T is saved with `lastRejectedAt` set
- When the client sends `PUT /api/v1/settings` with `leboncoinSession: {value: W9000, revision: r}` and no `claudeToken` part
- Then the answer is `200` (not `400 INVALID_SESSION`); the response and a following load show the session saved with hint `9f3a`, revision r, the same `updatedAt`, `status` and `lastAttempt`, and the token hint `wxyz` with its `lastRejectedAt`; zero Claude verification calls; a LeBoncoin check run afterwards sends `datadome=S`.
- Corner cases:
  - `Wmix`, `W8193` and `W8192` instead of `W9000`: same result;
  - the same save repeated three times: each `200`, revision still r;
  - session `expired` (with `expiresAt`) or `revoked`: the state and dates are kept, no reactivation;
  - no session saved: `200`, still no session saved;
  - the request body exceeds the save request size limit (for example 40 000 spaces): `413 REQUEST_TOO_LARGE`, nothing saved (S and T unchanged);
  - the response carries neither S nor T (#265 convention: no 5-character substring).

### TC-003 — Long whitespace-only Claude input, and mixing with real changes

- Covers: FR-SECRET-014, FR-SECRET-004, FR-CLT-SET-004; UC-SECRET-007 (alternatives)
- Level: integration (Go HTTP handlers, real SQLite, fake Claude)
- Given S is saved at revision r and T is saved with `lastRejectedAt` set
- When the client saves `claudeToken: {value: C1025}` with no session part
- Then `200` (not `400 INVALID_CLAUDE_TOKEN`); T is untouched with its `lastRejectedAt`; zero verification calls; S is untouched.
- Corner cases:
  - `C2000` and a mix of spaces, tabs and line breaks over 1024 characters: same result;
  - both parts long whitespace only (`W9000` at revision r and `C2000`): `200`, nothing changes, zero verification calls;
  - `W9000` session with token U (Claude accepts): one verification call with U, U is stored and `lastRejectedAt` cleared, S untouched at revision r;
  - `W9000` session with token U while Claude is unreachable: `502 CLAUDE_UNREACHABLE`, nothing saved;
  - `C2000` token with session `datadome=N` at revision r: N stored with revision r+1, T untouched, zero verification calls;
  - `C2000` token with an invalid session (`a=1; b=2`): `400 INVALID_SESSION`, nothing saved;
  - 1025 non-space characters: still `400 INVALID_CLAUDE_TOKEN` (too long), nothing saved.

### TC-004 — A long whitespace-only session input is never checked against the revision and keeps it

- Covers: FR-SECRET-014, FR-SECRET-008; UC-SECRET-007 (alternatives)
- Level: integration (Go HTTP handlers, real SQLite)
- Given modal A and modal B loaded Settings at revision r with session X, then the collector stored the renewed value Y (revision r+1)
- When modal A saves `leboncoinSession: {value: W9000, revision: r}`
- Then the answer is `200` without `SESSION_CHANGED`, and the session stays Y at revision r+1 with its state.
- And, in a second run without the renewal (both modals at revision r, session S): after modal A's `W9000` save, modal B replaces the session with N at revision r and succeeds without `SESSION_CHANGED`; a removal by modal B at revision r also succeeds.
- Corner cases:
  - modal A's `W9000` save together with token U (Claude accepts) after the renewal: `200`, U stored, session still Y at revision r+1.

## Settings modal

### TC-005 — Pasting a long blank selection into the LeBoncoin input saves without a "too long" message

- Covers: FR-SECRET-014, FR-LBC-SET-006, FR-CLT-SET-004; UC-SECRET-007 (main flow)
- Level: e2e (Playwright, mocked API)
- Given both secrets are saved (hints `wxyz`, `9f3a`, session revision 7) and the mocked save answers `200` with the same masked state
- When the operator fills the LeBoncoin input with `W9000`, leaves the Claude input empty, requests no removal and selects Save
- Then no request contains a session replacement or removal (the session part is absent, or the request is the untouched shape of the contract), and none contains 9000 spaces; no "too long" or other message is shown; the modal closes; reopening shows empty inputs and the hints `9f3a` and `wxyz` with their warnings unchanged.
- Corner cases:
  - `C1025` in the Claude input with the LeBoncoin input empty: no token replacement is sent, no progress for a verification, no message, the modal closes;
  - both inputs filled with long whitespace: Save changes nothing and closes, as with both inputs empty;
  - `W9000` in the LeBoncoin input with U in the Claude input: one request replacing only the token, no session part;
  - an input longer than 8192 characters that is not whitespace only, answered by the mock with `400 INVALID_SESSION` "The pasted text is too long (maximum 8192 characters).": the message is shown on the LeBoncoin entry, the modal stays open with the input kept;
  - Cancel after filling `W9000`: no request; reopening shows an empty input.

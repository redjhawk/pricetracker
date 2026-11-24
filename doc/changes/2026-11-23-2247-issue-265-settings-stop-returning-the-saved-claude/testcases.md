# Test cases: masked secrets in Settings (issue #265)

Sources: functional specification [settings-secret-masking](../../specifications/settings-secret-masking/functional.md) (FR-SECRET-001 to FR-SECRET-013), the amended rows of [LeBoncoin session settings](../../specifications/leboncoin-session-settings/functional.md) (FR-LBC-SET-001, 004, 008, 015, 016, 017, 018), [Claude token settings](../../specifications/claude-token-settings/functional.md) (FR-CLT-SET-003, 004, 005, 007, 008) and [session capture](../../specifications/leboncoin-session-capture/functional.md) (FR-LBC-CAP-013), and use cases [UC-SECRET-001 to UC-SECRET-006](../../use-cases/settings-secret-masking/).

Conventions:

- **Secrets:** T = `sk-ant-oat01-synthetic-token-wxyz` (saved Claude token), U = `sk-ant-oat01-synthetic-new-uvst` (new token), S = `synthetic-session-value-9f3a` (saved LeBoncoin session), N = `synthetic-renewed-or-new-n0p1` (new session), Y = `synthetic-renewed-value-y7k2` (session stored by a renewal). All are longer than the hint threshold. A **short secret** is shorter than the threshold set by the technical specification (for example `abc`). "Hint characters" are the last 4 characters (`wxyz`, `9f3a`, …).
- **No secret in a response** means: the body and headers contain neither the full secret nor any substring of it of 5 characters or more (checked for every 5-character window of the secret), and contain no character of a short secret.
- **Untouched, replace, remove:** the request shapes that express "keep", "replace with this input" and "remove" for each entry are those defined in `API_SPECIFICATION.md` by the technical specification. Tests use those shapes; they never send the hint as a value.
- **Go integration tests** (`internal/httpapi`, `internal/service`) use `newSettingsServer` / `newSessionService` (real SQLite database, normal migrations) and the fake Claude transport (`withClaudeStatus`) or `fakeClaude`, which count verification calls. "Claude unreachable" means the fake transport returns a network error. The collector's renewal, expiry, revocation and attempt recording are produced with the existing store functions used by the collection tests. Server logs are captured with `log.SetOutput`.
- **E2E tests** (Playwright, `tests/claude-token-settings.spec.ts` and `tests/leboncoin-session-settings.spec.ts`, adapted) mock the API with `page.route`/`context.route` as the existing specs do, record every request body, and answer with the masked shapes of the new contract. Existing e2e tests that expect the full value in an input or "saving an empty field clears the session" are rewritten by these cases.

## Coverage summary

| Requirement | Test cases |
| --- | --- |
| FR-CLT-SET-003 | TC-001, TC-012 |
| FR-CLT-SET-004 | TC-006, TC-013 |
| FR-CLT-SET-005 | TC-007, TC-014 |
| FR-CLT-SET-007 | TC-010 |
| FR-CLT-SET-008 | TC-002, TC-017 |
| FR-LBC-CAP-013 | TC-019 |
| FR-LBC-SET-001 | TC-012 |
| FR-LBC-SET-004 | TC-001, TC-012 |
| FR-LBC-SET-008 | TC-007, TC-014 |
| FR-LBC-SET-015 | TC-009, TC-016 |
| FR-LBC-SET-016 | TC-009, TC-016 |
| FR-LBC-SET-017 | TC-004, TC-008, TC-015 |
| FR-LBC-SET-018 | TC-002, TC-017 |
| FR-SECRET-001 | TC-001, TC-002 |
| FR-SECRET-002 | TC-001, TC-011, TC-012 |
| FR-SECRET-003 | TC-012 |
| FR-SECRET-004 | TC-003, TC-004, TC-005, TC-013 |
| FR-SECRET-005 | TC-006, TC-013 |
| FR-SECRET-006 | TC-007, TC-014 |
| FR-SECRET-007 | TC-005, TC-013 |
| FR-SECRET-008 | TC-008, TC-015 |
| FR-SECRET-009 | TC-009, TC-016 |
| FR-SECRET-010 | TC-010 |
| FR-SECRET-011 | TC-017 |
| FR-SECRET-012 | TC-019 |
| FR-SECRET-013 | TC-018 |

| Use case | Test cases |
| --- | --- |
| UC-SECRET-001 | TC-001, TC-002, TC-009, TC-010, TC-011, TC-012, TC-016, TC-017, TC-018 |
| UC-SECRET-002 | TC-002, TC-003, TC-005, TC-006, TC-013, TC-017 |
| UC-SECRET-003 | TC-002, TC-004, TC-005, TC-006, TC-008, TC-013, TC-017 |
| UC-SECRET-004 | TC-007, TC-014, TC-018 |
| UC-SECRET-005 | TC-008, TC-009, TC-015 |
| UC-SECRET-006 | TC-019 |

## Server

### TC-001 — Loading Settings returns whether each secret is saved and its hint, never the secret

- Covers: FR-SECRET-001, FR-SECRET-002, FR-CLT-SET-003, FR-LBC-SET-004; UC-SECRET-001
- Level: integration (Go HTTP handlers, real SQLite)
- Given T and S are saved
- When the client loads Settings (`GET /api/v1/settings/claude-token` and `GET /api/v1/settings/leboncoin-session`, or the load endpoint the technical specification defines)
- Then each answer is `200` with `Cache-Control: no-store`, states that a secret is saved and gives the hint characters `wxyz` and `9f3a`, carries no secret (convention), and keeps the other fields (session `revision`, `status`, `updatedAt`, `expiresAt`, `revokedAt`, `lastAttempt`; token `updatedAt`, `lastRejectedAt`).
- Corner cases:
  - nothing saved: each answer states that no secret is saved and gives no hint characters;
  - short secret (session `abc`, saved directly through the store since it is a valid cookie value): saved, no hint characters, no character of it in the body;
  - secret exactly at the threshold length: the hint follows the threshold rule chosen by the technical specification (no characters below it, last 4 from it on);
  - session saved from a cookie string `Cookie: a=1; datadome=S; b=2`: the hint is that of S, and neither `a=1` nor `b=2` appears;
  - protected mode: alice (T, S saved) and bob (other secrets) each receive only their own hints, and neither body contains the other's secrets;
  - repeated loads give identical answers and change nothing in the database.

### TC-002 — No response or log sends a saved or typed secret back

- Covers: FR-SECRET-001, FR-CLT-SET-008, FR-LBC-SET-018; UC-SECRET-001, UC-SECRET-002, UC-SECRET-003
- Level: integration (Go HTTP handlers, real SQLite, captured logs)
- Given T and S are saved, and a LeBoncoin item with a recorded session-assisted attempt and an AI review exist
- When the client calls every settings route (load, each successful save shape: replace session with N, replace token with U, remove each, both untouched), every error path of the save (`INVALID_JSON`, `INVALID_REQUEST`, `INVALID_SESSION`, `INVALID_CLAUDE_TOKEN`, `CLAUDE_TOKEN_REJECTED`, `CLAUDE_UNREACHABLE`, `SESSION_CHANGED`, `REQUEST_TOO_LARGE`, `METHOD_NOT_ALLOWED`), `PUT /api/v1/settings/leboncoin-session` if the technical specification keeps it, the item list and details, the application information route and the auth routes
- Then no response carries T, S, U or N (convention); successful save responses carry only the saved state and hints of the stored secrets; the captured server log contains none of them.
- Corner cases:
  - a refused token U (`CLAUDE_TOKEN_REJECTED`) and an invalid session input containing `datadome=` with an invalid value: the error message does not echo the typed input;
  - a save refused with `SESSION_CHANGED` after a renewal to Y: the response carries no Y;
  - storage failure on load (`500 INTERNAL_ERROR`): no secret in the body.

### TC-003 — Saving only the LeBoncoin session leaves the Claude token exactly as it is

- Covers: FR-SECRET-004; UC-SECRET-002
- Level: integration (Go HTTP handlers, real SQLite, fake Claude)
- Given T is saved with a `lastRejectedAt` set (rejected-token warning), S is saved, and the fake Claude is unreachable
- When the client saves a replacement of the session with `datadome=N` and the Claude entry untouched
- Then the save succeeds; zero verification calls were made; the stored token is still T with the same `updatedAt` and `lastRejectedAt`; the stored session is N with a new revision; reloading shows the token hint `wxyz` and its warning unchanged.
- Corner cases:
  - Claude entry sent as whitespace only (`"   "`) if the contract maps it to untouched: same result, never a removal;
  - Claude answers 401 instead of being unreachable: same result (not called);
  - an AI review started after the save sends T to Claude (fake records the token).

### TC-004 — Saving only the Claude token leaves the LeBoncoin session exactly as it is

- Covers: FR-SECRET-004, FR-LBC-SET-017; UC-SECRET-003
- Level: integration (Go HTTP handlers, real SQLite, fake Claude accepting)
- Given S is saved at revision r, with a recorded `lastAttempt` (`rejected`) and, in a second run, `status: "expired"` with `expiresAt`; a second client ("modal B") has loaded Settings at revision r
- When the first client saves a replacement of the token with U and the session untouched
- Then the save succeeds; one verification call was made with U; the stored session keeps its value S, revision r, `updatedAt`, `status`, `expiresAt`, `revokedAt` and `lastAttempt`; and modal B then replaces the session with N at revision r and succeeds without `SESSION_CHANGED`.
- Corner cases:
  - the Claude verification fails (`CLAUDE_UNREACHABLE`): nothing changes, session revision still r;
  - a LeBoncoin check started after the save sends `datadome=S` (expired case: no session is sent, as before).

### TC-005 — Untouched saves never change or replace a secret

- Covers: FR-SECRET-004, FR-SECRET-007; UC-SECRET-002, UC-SECRET-003
- Level: integration (Go HTTP handlers, real SQLite, fake Claude)
- Given T and S are saved
- When the client repeats three times: load Settings, then save with both entries untouched (the contract's shape, or no request if the contract defines that Save sends nothing)
- Then each save succeeds; the stored token is T and the session S, with unchanged revision, `updatedAt`, `lastRejectedAt` and `lastAttempt`; no verification call was made; the hints are still `wxyz` and `9f3a`.
- Corner cases:
  - afterwards a LeBoncoin check sends `datadome=S` and an AI review sends T (secrets not replaced by their hints);
  - a client that sends the hint as a replacement value (`wxyz` for the session, or the masked hint text) is treated as a replacement by the server, so this case asserts the frontend never sends it (TC-013); the server test asserts the masked hint text, if it contains mask characters, is rejected as `INVALID_SESSION` / `INVALID_CLAUDE_TOKEN` and saves nothing;
  - no secret saved and both untouched: success, nothing created.

### TC-006 — A non-empty input is always a replacement

- Covers: FR-SECRET-005, FR-CLT-SET-004; UC-SECRET-002, UC-SECRET-003
- Level: integration (Go service and HTTP handlers, real SQLite, fake Claude)
- Given T is saved with `lastRejectedAt` set and S is saved with `lastAttempt` `rejected`
- When the client saves a replacement of the token with T itself (same value) and of the session with S itself
- Then Claude is called once with T; `lastRejectedAt` is cleared; the session revision increases, `lastAttempt` is `null` and `status` is `active`; the hints are unchanged.
- Corner cases:
  - token `  U  ` is trimmed and stored as U; `sk ant`, a line break, a non-ASCII character or 1025 characters give `INVALID_CLAUDE_TOKEN` and save nothing;
  - Claude refuses U (`CLAUDE_TOKEN_REJECTED`) or is unreachable while the session is replaced in the same request: neither entry is saved;
  - session input `Cookie: a=1; datadome=N; b=2` stores N and the hint shows N's last 4 characters; `a=1; b=2` gives `INVALID_SESSION` and saves nothing;
  - a revoked session replaced with N becomes `active` with `revokedAt: null`.

### TC-007 — Removal is explicit; an empty input never removes

- Covers: FR-SECRET-006, FR-CLT-SET-005, FR-LBC-SET-008, FR-SECRET-004; UC-SECRET-004
- Level: integration (Go HTTP handlers, real SQLite, fake Claude unreachable)
- Given T and S are saved at session revision r
- When the client saves a removal of the token with the session untouched, then a removal of the session at revision r with the token untouched
- Then the token removal succeeds with zero verification calls and leaves S unchanged; the session removal succeeds; reloading states that neither secret is saved; a following LeBoncoin check sends no session; the item AI review reports that a Claude token must be configured.
- Corner cases:
  - empty string and whitespace-only inputs for each entry keep the secret (untouched) and never remove it;
  - removing a secret that is not saved: success, nothing changes (or the error the contract defines), and no other entry changes;
  - one request removing the token and replacing the session with N: both applied; the same request with an invalid session input saves neither (token still T);
  - one request removing the session and replacing the token with a refused U: neither applied;
  - removal and replacement of the same entry in one request (if the contract makes it expressible): refused with the error the contract defines, nothing saved.

### TC-008 — The "session changed" check applies only when the session is replaced or removed

- Covers: FR-SECRET-008, FR-LBC-SET-017; UC-SECRET-005, UC-SECRET-003
- Level: integration (Go HTTP handlers, real SQLite, fake Claude accepting)
- Given the client loaded Settings at revision r with session X, then the collector stored the renewed value Y (revision r+1)
- When the client, with revision r, (a) replaces the session with N, (b) removes the session, (c) replaces the token with U and leaves the session untouched
- Then (a) and (b) answer `409 SESSION_CHANGED` with the existing message and save nothing (a token replacement in the same request is not stored either); (c) succeeds and the session stays Y at revision r+1 with its state.
- And after reloading (revision r+1, hint of Y), replacing with N and removing both succeed.
- Corner cases:
  - the stored session changed through revocation, through a save from another client, or through a removal from another client: same results;
  - expiry passing without a revision change: no `SESSION_CHANGED` (unchanged rule);
  - repeated refused saves keep the stored state Y.

### TC-009 — Renewed, expired and revoked sessions are shown by their hint and state

- Covers: FR-SECRET-009, FR-LBC-SET-015, FR-LBC-SET-016; UC-SECRET-001, UC-SECRET-005
- Level: integration (Go HTTP handlers, real SQLite)
- Given S is saved
- When the collector stores the renewed value Y, and the client reloads Settings; then LeBoncoin's expiry passes, reload; then LeBoncoin revokes the session, reload
- Then the first load shows the hint of Y (`y7k2`) and `status: "active"`; the second `status: "expired"` with the hint of the stored value; the third `status: "revoked"` with the hint if a value remains, or "no session saved" if none; no load carries S or Y.
- Corner cases:
  - a token replaced or removed from another client: reloading shows the new token hint or "no token saved".

### TC-010 — Secrets saved before this change keep working and survive a restart

- Covers: FR-SECRET-010, FR-CLT-SET-007; UC-SECRET-001
- Level: integration (Go HTTP handlers, real SQLite file)
- Given a database file in which T and S were stored as the previous version stores them (rows written through the existing store functions before any new migration, or a fixture database of the previous schema)
- When the server starts on that file, the client loads Settings, a LeBoncoin check and an AI review run, then the server is stopped and started again on the same file and Settings is loaded again
- Then both loads show hints `wxyz` and `9f3a`; the LeBoncoin check sends `datadome=S`; the fake Claude receives T; no save was needed.
- Corner cases:
  - a previously saved short secret: shown as saved without characters, still used;
  - previous rejected-token warning and session `lastAttempt` are kept.

### TC-011 — The hint function

- Covers: FR-SECRET-002; UC-SECRET-001
- Level: unit (Go)
- Given the function that builds a hint from a stored secret
- When it is called with T, S, a value exactly at the threshold, a value one character below it, a 1-character value and no value
- Then it returns "saved, ends with `wxyz`" for T, `9f3a` for S, the threshold result chosen by the technical specification for the boundary values, "saved, no characters" below the threshold, and "not saved" for no value; it never returns more than 4 characters of the secret.

## Settings modal

### TC-012 — Settings shows hints, empty inputs and the new help texts

- Covers: FR-SECRET-002, FR-SECRET-003, FR-CLT-SET-003, FR-LBC-SET-001, FR-LBC-SET-004; UC-SECRET-001
- Level: e2e (Playwright, mocked API)
- Given the API reports both secrets saved with hints `wxyz` and `9f3a`
- When the operator opens Settings
- Then the Claude entry states, in text outside its input, that a token is saved ending in `wxyz`; the LeBoncoin entry that a session is saved ending in `9f3a`; both inputs are empty; each help text says what may be pasted and that leaving the input empty keeps the saved secret; no text says that saving an empty field removes it; "Remove token" and "Remove session" are offered.
- Corner cases:
  - nothing saved: each entry states that none is saved, the Claude help still says AI reviews are unavailable until a token is saved, and no remove action is shown;
  - short secret (saved, no hint characters): the entry states it is saved and shows no characters;
  - loading and load failure: no hint, input, remove action or Save is usable; Cancel works and reopening retries;
  - reopening after a cancel shows empty inputs again.

### TC-013 — Save sends only replacements; untouched entries and hints are never sent

- Covers: FR-SECRET-004, FR-SECRET-005, FR-SECRET-007, FR-CLT-SET-004; UC-SECRET-002, UC-SECRET-003
- Level: e2e (Playwright, mocked API)
- Given both secrets are saved (hints `wxyz`, `9f3a`, session revision 7)
- When the operator pastes `datadome=N` into the LeBoncoin input, leaves the Claude input empty and saves
- Then exactly one save request is sent: it replaces the session with the typed text and revision 7, marks the Claude entry untouched (per the contract), and contains neither `wxyz` nor `9f3a` nor the masked hint text; the modal closes; reopening shows empty inputs and the new hint returned by the server.
- Corner cases:
  - the reverse: a token U typed, session empty: the request replaces the token and leaves the session untouched (no session change, so no session revision check is requested); Save shows progress while verifying;
  - both inputs empty, no removal: no secret, no hint and no removal in any request (or no request, per the contract); the modal closes; reopening shows the same hints and warnings;
  - whitespace-only input: treated as untouched;
  - the same token as the saved one pasted again: sent as a replacement;
  - `INVALID_SESSION`, `INVALID_CLAUDE_TOKEN`, `CLAUDE_TOKEN_REJECTED` or a server error: the modal stays open with the typed input and the message on the right entry; a retry sends the same request;
  - pressing Save repeatedly sends at most one request.

### TC-014 — Remove actions mark a secret for removal, with undo and cancel

- Covers: FR-SECRET-006, FR-CLT-SET-005, FR-LBC-SET-008; UC-SECRET-004
- Level: e2e (Playwright, mocked API)
- Given both secrets are saved
- When the operator selects "Remove token", leaves the session input empty and saves
- Then before saving the Claude entry states in text that the token will be removed on Save, its input cannot be used and an undo action is offered; the request removes the token and leaves the session untouched; the modal closes; reopening (API now reports no token) states that no token is saved with no remove action.
- Corner cases:
  - "Remove session", then undo, then Save with an empty input: the request contains no removal and no session change; the input is usable again after undo;
  - "Remove session" then Cancel or Escape: no request; reopening shows the session hint and no pending removal;
  - remove the token and paste a new session in one Save: one request with both;
  - save refused (`SESSION_CHANGED`, `INVALID_SESSION` on the other entry, server error): the pending removal and its text are kept for a retry;
  - typing into an input and then choosing remove for that entry: the typed text is not sent.

### TC-015 — "Session changed" warning when replacing or removing a changed session

- Covers: FR-SECRET-008, FR-LBC-SET-017; UC-SECRET-005
- Level: e2e (Playwright, mocked API)
- Given Settings loaded at revision 7 and the mocked save answers `409 SESSION_CHANGED`
- When the operator pastes N (or chooses "Remove session") and saves
- Then the modal stays open with the input (or the pending removal) and the warning that the session changed and Settings must be reopened; after reopening (API now at revision 8 with hint `y7k2`), the hint shows `y7k2` and the same save succeeds.
- Corner cases:
  - only a token typed: the request does not ask for a session check and the mocked server answers `200`; no warning is shown.

### TC-016 — Warnings and advice next to the hints

- Covers: FR-SECRET-009, FR-LBC-SET-015, FR-LBC-SET-016; UC-SECRET-001
- Level: e2e (Playwright, mocked API)
- Given the API reports, in successive openings: a session renewed to Y (hint `y7k2`), an expired session with a hint, a revoked session with no remaining value, a session whose latest attempt was rejected, and a token with `lastRejectedAt` set
- When the operator opens Settings each time
- Then the hint matches the stored secret (`y7k2` after renewal); the expired/revoked warning says the session is not used and advises capturing a new session or removing it, without mentioning saving an empty field; the rejected-session and rejected-token warnings are shown beside the hints as before.
- Corner cases:
  - after a successful replacement the reopened modal (API reports the new secret with no warning) shows no warning.

### TC-017 — The browser holds no full secret after the modal closes

- Covers: FR-SECRET-011, FR-CLT-SET-008, FR-LBC-SET-018; UC-SECRET-001, UC-SECRET-002, UC-SECRET-003
- Level: e2e (Playwright, mocked API)
- Given both secrets are saved
- When the operator types U and N, saves (success), reopens, types again and cancels, then types again, gets a save error and closes
- Then after each close `localStorage`, `sessionStorage`, IndexedDB, cookies and the page's DOM and URL contain neither U nor N; every reopening shows empty inputs; no API response used by the page contains a secret (the mocks return only hints, and the page never requests a route that would return one).
- Corner cases:
  - a narrow viewport (400 px) and a reload of the page give the same result.

### TC-018 — Hints and remove actions are accessible and fit a narrow screen

- Covers: FR-SECRET-013; UC-SECRET-001, UC-SECRET-004
- Level: e2e (Playwright, mocked API)
- Given both secrets are saved
- When the operator uses only the keyboard, then the viewport is about 400 px wide
- Then each hint is text associated with its input (exposed in the input's accessible description); Tab reaches "Remove token" and "Remove session", whose accessible names name the secret, and Enter or Space triggers them; the pending-removal text is announced and associated with its entry; the undo action is reachable and named for its secret; focus stays in the modal and Escape still closes it; at 400 px the hints, actions and inputs are visible without horizontal page scrolling.
- Corner cases:
  - after undo, focus is on a control of the same entry, not lost to the page;
  - the pending removal is conveyed by text, not only by colour.

## Documentation

### TC-019 — The operator guide describes hints, keeping and removing

- Covers: FR-SECRET-012, FR-LBC-CAP-013; UC-SECRET-006
- Level: unit (documentation test, Go, reading `doc/leboncoin-session.md`, with in-test fixtures as in `internal/service/amazon_browser_snap_docs_test.go`)
- Given the operator guide
- When the test reads it
- Then its Settings instructions say that Settings shows a masked hint (the last 4 characters) instead of the session, that leaving the input empty keeps the saved session, that pasting a new value replaces it, and that the session is cleared with "Remove session" then Save; it no longer says to clear by saving an empty field, nor that anyone can read the value in Settings; it still describes the paste-into-Settings workflow, the check and renewal loop, the secret-handling warning, and contains no `scp`, systemd drop-in, environment variable or session file instructions.
- Corner cases (on fixtures):
  - fixture still saying "saving an empty **LeBonCoin session** field": fails;
  - fixture without "Remove session": fails;
  - fixture with the remove instruction only inside a fenced code block: fails;
  - fixture of the guide as the technical specification describes it: passes.

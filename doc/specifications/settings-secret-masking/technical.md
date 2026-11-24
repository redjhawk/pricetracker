# Technical specification: masked secrets in Settings

Status: ready (issue #265; TS-SECRET-017–019 added by issue #286)
Functional specification: [functional.md](functional.md), FR-SECRET-001–014; amended rows of [LeBoncoin session settings](../leboncoin-session-settings/functional.md) (FR-LBC-SET-001, 004, 008, 015–018), [Claude token settings](../claude-token-settings/functional.md) (FR-CLT-SET-003–005, 007, 008) and [session capture](../leboncoin-session-capture/functional.md) (FR-LBC-CAP-013)
Test cases: [testcases.md](../../changes/2026-11-23-2247-issue-265-settings-stop-returning-the-saved-claude/testcases.md), TC-001–019 (issue #265); [testcases.md](../../changes/2026-11-24-0036-issue-286-treat-long-whitespace-only-leboncoin-ses/testcases.md), TC-001–005 (issue #286)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), sections "Secret hint (issue #265)", "LeBoncoin session input", "Claude token object" and "Save settings"

## Summary

The server stops serializing the stored secrets. Both settings objects carry `saved` and `hint` (last 4 characters when the secret has at least 16 characters, else `null`) instead of `value`. `PUT /api/v1/settings` treats an absent part or an empty `value` as "untouched", a non-empty `value` as "replace" and `"remove": true` as "remove"; the LeBoncoin revision is checked only on replace or remove. The legacy `PUT /api/v1/settings/leboncoin-session` is removed. The modal shows the hint as text, always starts with empty inputs, sends only replacements and removals, and offers a Remove/Undo toggle per saved secret. No schema change.

## Requirement mapping

`SECRET-*` abbreviates `FR-SECRET-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-SECRET-001 | SECRET-002 | Hint function `model.SecretHint(value *string) *string`, threshold constant `model.SecretHintMinLength = 16` | `internal/model/model.go` | TC-011 |
| TS-SECRET-002 | SECRET-001, SECRET-002, SECRET-010, CLT-SET-003, CLT-SET-007, LBC-SET-004, LBC-SET-016 | `Value` fields become `json:"-"`; new `Saved`/`Hint` fields filled by the store's row mapping on every read | `internal/model/model.go`, `internal/store/store.go`, `internal/store/claude_token.go` | TC-001, TC-009, TC-010 |
| TS-SECRET-003 | SECRET-004–008, CLT-SET-004, CLT-SET-005, LBC-SET-008, LBC-SET-017 | New `PUT /api/v1/settings` request shape (absent / `value` / `remove`) decoded by the handler | `internal/httpapi/settings.go` | TC-002–008 |
| TS-SECRET-004 | SECRET-004–008, CLT-SET-004, CLT-SET-005, LBC-SET-017 | Service `SaveSettings` rules: untouched → no parse, no verification, no revision check; non-empty token always verified; removal never verified | `internal/service/settings.go` | TC-003–008 |
| TS-SECRET-005 | SECRET-001, SECRET-006 | Remove `PUT /api/v1/settings/leboncoin-session` (405, `Allow: GET`) and the service `SaveLeboncoinSession` | `internal/httpapi/server.go`, `internal/service/service.go` | TC-002 |
| TS-SECRET-006 | SECRET-001, CLT-SET-008, LBC-SET-018 | No secret in any response, error message or log (fixed messages, `json:"-"`, no request-body logging) | handlers, service | TC-002 |
| TS-SECRET-007 | SECRET-010, CLT-SET-007 | No migration: hints are computed from the stored values at read time | none | TC-010 |
| TS-SECRET-008 | SECRET-001, SECRET-007 | Frontend API types `saved`/`hint`; `SettingsChange` union with `remove`; `saveLeboncoinSession` deleted | `src/api/settings.ts` | `npm run build`; TC-013 |
| TS-SECRET-009 | SECRET-002, SECRET-003, CLT-SET-003, LBC-SET-001, LBC-SET-004 | Modal: state line (hint / none / pending removal) in each entry's helper text, empty inputs, new help texts | `src/components/SettingsModal.tsx` | TC-012 |
| TS-SECRET-010 | SECRET-004, SECRET-005, SECRET-007, CLT-SET-004 | Save sends only non-blank inputs and removals; no request when both untouched | `src/components/SettingsModal.tsx` | TC-013 |
| TS-SECRET-011 | SECRET-006, CLT-SET-005, LBC-SET-008 | Remove/Undo toggle button per saved secret | `src/components/SettingsModal.tsx`, `src/index.css` (only if the layout needs it) | TC-014 |
| TS-SECRET-012 | SECRET-008, LBC-SET-017 | `SESSION_CHANGED` keeps inputs and pending removals; only sent when the session is replaced or removed | `src/components/SettingsModal.tsx` | TC-015 |
| TS-SECRET-013 | SECRET-009, LBC-SET-015, LBC-SET-016 | Expired/revoked advice says to remove the session | `src/components/SettingsModal.tsx` | TC-016 |
| TS-SECRET-014 | SECRET-011, CLT-SET-008, LBC-SET-018 | Typed inputs and pending removals reset on close; no storage | `src/components/SettingsModal.tsx` | TC-017 |
| TS-SECRET-015 | SECRET-013 | Hint and pending text in the input's accessible description, polite live region, named toggle buttons | `src/components/SettingsModal.tsx` | TC-018 |
| TS-SECRET-016 | SECRET-012, LBC-CAP-013 | Operator guide describes the hint, keeping and "Remove session" (updated in this technical stage) | `doc/leboncoin-session.md` | TC-019 |
| TS-SECRET-017 | SECRET-014, SECRET-004, SECRET-008, LBC-SET-006 | `sessionChangeFor` returns "untouched" for a whitespace-only `value` before `ParseSessionInput`, so the 8192-character input limit applies only to input that is not whitespace only (issue #286) | `internal/service/settings.go`, `internal/leboncoin/session.go` (doc comment only) | TC-001, TC-002, TC-004 of #286 |
| TS-SECRET-018 | SECRET-014, SECRET-004, CLT-SET-004 | Claude token: already trimmed before the 1024-character check, so a whitespace-only token of any length is untouched; no code change | none | TC-003 of #286 |
| TS-SECRET-019 | SECRET-014, LBC-SET-006, CLT-SET-004 | Frontend already sends no part for a blank input (`trim()`) and has no length check; no code change | none | TC-005 of #286 |

## API

Contract: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md). Changes, all in this release (the embedded frontend is the only client):

- `GET /api/v1/settings/leboncoin-session` → `{"session": {"saved", "hint", "revision", "updatedAt", "status", "expiresAt", "revokedAt", "lastAttempt"}}`; any other method `405` with `Allow: GET`.
- `GET /api/v1/settings/claude-token` → `{"claudeToken": {"saved", "hint", "updatedAt", "lastRejectedAt"}}`.
- `PUT /api/v1/settings`: per part, absent = untouched; `{"value": "<non-blank>"}` = replace; `{"value": "<blank>"}` = untouched; `{"remove": true}` = remove. `leboncoinSession` always requires `revision` (checked only on replace/remove). `{}` → `200` with the current objects. Neither or both of `value`/`remove`, or `remove` ≠ `true` → `400 INVALID_REQUEST`. Response masked like the GETs. Other error codes and messages unchanged.
- Hint rule: `hint` = last 4 characters (runes) when the stored value has at least 16 characters; otherwise `null`. Secrets are ASCII by validation, so characters equal bytes in practice.
- Mask text pasted back (for example `••••wxyz`) contains non-ASCII characters and is rejected by the existing validators (`INVALID_SESSION`, `INVALID_CLAUDE_TOKEN`); a bare 4-character hint is a valid replacement value, so the frontend must never send it (TC-005, TC-013).

## Backend

### Model and hint (TS-SECRET-001, TS-SECRET-002)

`internal/model/model.go`:

```go
// SecretHintMinLength is the shortest saved secret whose last 4 characters are shown.
const SecretHintMinLength = 16

// SecretHint returns the last 4 characters of a saved secret, or nil when none is
// saved or it is shorter than SecretHintMinLength.
func SecretHint(value *string) *string

type LeboncoinSession struct {
	Value       *string `json:"-"`     // never serialized (issue #265)
	Saved       bool    `json:"saved"`
	Hint        *string `json:"hint"`
	Revision    int64   `json:"revision"`
	// UpdatedAt, Status, ExpiresAt, RevokedAt, LastAttempt unchanged
}

type ClaudeToken struct {
	Value          *string    `json:"-"`
	Saved          bool       `json:"saved"`
	Hint           *string    `json:"hint"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	LastRejectedAt *time.Time `json:"lastRejectedAt"`
	Revision       int64      `json:"-"`
	OwnerID        int64      `json:"-"`
}
```

`SecretHint` counts runes (`[]rune`), returns `nil` for `nil` or fewer than 16 runes, else a new string of the last 4 runes. Field order fixes the JSON order (`saved`, `hint` first) for tests comparing whole bodies.

The store fills `Saved = Value != nil` and `Hint = model.SecretHint(Value)` in `leboncoinSessionRow.model` and `readClaudeToken`, the only constructors of these objects for responses (GET and `SaveSettings` results). `Value` stays in the structs for the collector (`service.go`) and AI reviews (`review.go`), which are unchanged. `json:"-"` guarantees that no handler can serialize a secret by accident.

### HTTP (TS-SECRET-003, TS-SECRET-005, TS-SECRET-006)

`internal/httpapi/settings.go`, `handleSettings` decodes:

```go
var body struct {
	LeboncoinSession *struct {
		Value    *string `json:"value"`
		Remove   *bool   `json:"remove"`
		Revision *int64  `json:"revision"`
	} `json:"leboncoinSession"`
	ClaudeToken *struct {
		Value  *string `json:"value"`
		Remove *bool   `json:"remove"`
	} `json:"claudeToken"`
}
```

- Size limit, `INVALID_JSON`, trailing data and `REQUEST_TOO_LARGE` handling unchanged.
- For each present part: exactly one of `Value` and `Remove` non-nil, `Remove` (if present) must be `true`; the session part needs `Revision != nil && *Revision >= 0`. Otherwise `400 INVALID_REQUEST` with the message "Each Settings entry sent must have either a value or \"remove\": true, and the LeBoncoin session needs the revision from Settings."
- A JSON `null` part decodes to nil and is treated as absent (same as today).
- The "no part present" error is dropped: `{}` reaches the service and returns the current state.
- The handler passes raw strings and the remove flags to the service; blank handling is a service rule.

`internal/httpapi/server.go`, `handleLeboncoinSession`: keep the `GET` branch; delete the `PUT` branch; default branch sets `Allow: GET`.

No new logging. `serverError` logs only store errors (parameterized SQL, no bound values). Validation and verification messages are fixed strings and never include the input.

### Service (TS-SECRET-004, TS-SECRET-005)

`internal/service/settings.go`:

```go
// SettingsInput holds the Settings entries to save; a nil part is left unchanged.
type SettingsInput struct {
	LeboncoinSession *SessionInput
	ClaudeToken      *TokenInput
}

// SessionInput replaces the LeBoncoin session with Value, or removes it, if the
// stored revision is still Revision. A blank Value leaves the session unchanged.
type SessionInput struct {
	Value    string
	Remove   bool
	Revision int64
}

// TokenInput replaces the Claude token with Value, or removes it. A blank Value
// leaves the token unchanged.
type TokenInput struct {
	Value  string
	Remove bool
}
```

`SaveSettings(ctx, input)` (first error wins, nothing saved):

1. Session: `Remove` → `sessionChange = &store.SessionChange{Value: nil, ExpectedRevision}`. Otherwise `leboncoin.ParseSessionInput(Value)`: `SessionInputError` → `400 INVALID_SESSION`; `""` (blank) → `sessionChange = nil` (untouched, no revision check); else `&store.SessionChange{Value: &value, ExpectedRevision}`.
2. Token: `Remove` → `tokenChange = &store.TokenChange{Value: nil}` (no verification). Otherwise trim; blank → `tokenChange = nil` (untouched); else `validateClaudeToken`, then `verifyClaudeToken` (always; the "equal to stored token" shortcut and its store read are deleted, FR-SECRET-005), then `&store.TokenChange{Value: &token}`.
3. `store.SaveSettings(ctx, owner, sessionChange, tokenChange, now)`; `ErrSessionChanged` → existing `409 SESSION_CHANGED`.

Verification still runs outside any transaction. `internal/service/service.go`: delete `SaveLeboncoinSession` (its only caller was the removed route); `LeboncoinSession` stays.

### Store and persistence (TS-SECRET-007)

No schema change and no migration: `leboncoin_session` and `claude_token` keep their columns; values saved before this change are read unchanged and masked at read time (FR-SECRET-010). `store.SaveSettings` already implements every case: nil part → untouched (session read for the response; token untouched); `SessionChange{Value: nil}` → `saveLeboncoinSessionTx` checks the revision, then removes (no change when already empty); `TokenChange{Value: nil}` → removes, no change when already empty; a non-nil token value replaces, increments the revision and clears `last_rejected_at`. `store.SaveLeboncoinSession` stays (store tests and test setup use it).

Concurrency is unchanged: one transaction per save; an untouched session is only read, so a concurrent renewal or revocation is kept (FR-SECRET-008); collector writes stay conditional on the revision captured at attempt start.

## Frontend

### API module (TS-SECRET-008) — `src/api/settings.ts`

```ts
export interface LeboncoinSessionSettings {
  saved: boolean;
  hint: string | null;
  revision: number;
  updatedAt: string | null;
  status: "none" | "active" | "expired" | "revoked";
  expiresAt: string | null;
  revokedAt: string | null;
  lastAttempt: { outcome: "accepted" | "rejected" | "failed"; attemptedAt: string } | null;
}
export interface ClaudeTokenSettings {
  saved: boolean;
  hint: string | null;
  updatedAt: string | null;
  lastRejectedAt: string | null;
}
export interface SettingsChange {
  leboncoinSession?: { value: string; revision: number } | { remove: true; revision: number };
  claudeToken?: { value: string } | { remove: true };
}
export function saveSettings(change: SettingsChange): Promise<{ session: LeboncoinSessionSettings; claudeToken: ClaudeTokenSettings }>;
```

`getLeboncoinSession` and `getClaudeToken` keep their paths. `saveLeboncoinSession` is deleted (route removed).

### Component — `src/components/SettingsModal.tsx`

New local state: `removeSession`, `removeToken` (booleans). `value` and `claudeValue` always start as `""` after a load (no longer seeded from the response). The cleanup on close resets both flags with the other state (FR-SECRET-011, TS-SECRET-014). Nothing is written to browser storage.

**State line and help (TS-SECRET-009, TS-SECRET-015).** Each entry's Carbon `helperText` is a node with two parts: a state line `<span className="settings-secret-state" aria-live="polite">` rendered as a block, followed by the help sentence. Carbon puts the helper text in the input's `aria-describedby` (both `TextArea` and `TextInput`, enabled or disabled), so the hint is the input's accessible description; the live region announces the change to and from the pending-removal text. While the entry shows a validation error Carbon replaces the helper with `invalidText` (existing behaviour).

| Entry state | LeBoncoin state line | Claude state line |
| --- | --- | --- |
| pending removal | "The saved session will be removed when you save." | "The saved token will be removed when you save." |
| `saved` with `hint` | "A session is saved: ••••{hint}." | "A token is saved: ••••{hint}." |
| `saved`, `hint === null` | "A session is saved." | "A token is saved." |
| not saved | "No session is saved." | "No token is saved." |

Help sentences (FR-SECRET-003; no text says that an empty save removes a secret):

- LeBoncoin: "Paste the datadome value or a cookie string containing datadome=…. Leave it empty to keep the saved session."
- Claude: "Paste the token printed by claude setup-token (Claude Pro/Max subscription). It is used for LeBoncoin AI reviews. Leave it empty to keep the saved token." plus " AI reviews are unavailable until a token is saved." when `!claudeToken.saved` (FR-CLT-SET-001/003/013).

The bullets are display text only; nothing in the component derives a request value from `hint`.

**Remove / Undo (TS-SECRET-011, TS-SECRET-015).** Under each entry, when the loaded object has `saved: true`, one Carbon `Button kind="ghost" size="sm"` toggles the pending removal. Labels: "Remove session" / "Undo session removal", "Remove token" / "Undo token removal" (the visible label is the accessible name and names the secret). The same element changes label, so focus stays on it after Remove and after Undo. Choosing Remove clears that entry's typed input and its `invalidText`; while pending, the input is `disabled`. The button is disabled while loading or saving and not rendered after a load failure or when nothing is saved. No colour-only signal: the pending state is the state-line text and the button label.

**Warnings (TS-SECRET-013).** Unchanged except the expired/revoked suffix, which becomes "It is no longer used. Capture a new session and save it here, or remove it." The rejected/failed session warnings and the rejected-token warning are unchanged and stay next to their entries.

**Save (TS-SECRET-010, TS-SECRET-012).**

```ts
const leboncoinSession = removeSession ? { remove: true as const, revision: settings.revision }
  : value.trim() ? { value, revision: settings.revision } : undefined;
const claudeToken = removeToken ? { remove: true as const }
  : claudeValue.trim() ? { value: claudeValue } : undefined;
if (!leboncoinSession && !claudeToken) { onClose(); return; } // both untouched: no request
await saveSettings({ ...(leboncoinSession ? { leboncoinSession } : {}), ...(claudeToken ? { claudeToken } : {}) });
```

- Inputs are sent as typed (the server trims). A typed text equal to the saved secret is still sent (FR-SECRET-005).
- The existing guards stay: `savingRef` (at most one request), Save disabled while loading, saving or after `SESSION_CHANGED`; close ignored while saving; "Saving…" and `InlineLoading` "Saving settings…".
- Error mapping unchanged (`INVALID_SESSION` on the session input, Claude codes on the token input, `SESSION_CHANGED` warning, other errors in "Could not save settings"). On any error the typed inputs and the pending removals are kept for a retry; retry sends the same request.
- `SESSION_CHANGED` can only follow a session replacement or removal, because an untouched session is not sent.
- Success closes the modal; reopening reloads both objects and shows empty inputs.

**UI states.** Loading and load failure unchanged: no state line, input, remove button or Save is usable (the existing `!loaded` guards also cover the new button). At about 400 px the `sm` modal is full width; the state line wraps; the ghost button sits on its own line. Add a rule to `src/index.css` only if the button needs spacing from the input (Carbon spacing tokens).

## Operator guide (TS-SECRET-016)

The developer stage cannot edit `doc/leboncoin-session.md`, so this technical stage updates it. "Paste it into Settings", "Check the result", "Renew or clear" and "Keep the value secret" now say that Settings shows a masked hint (the last 4 characters) instead of the session, that leaving the input empty keeps the saved session, that pasting a new value replaces it, and that the session is cleared with **Remove session** then **Save**. The guide no longer says to clear by saving an empty field or that anyone can read the value in Settings. The TC-019 test reads that file and uses in-test fixtures for its negative cases.

## Long whitespace-only input (issue #286)

Defect: `PUT /api/v1/settings` with `{"leboncoinSession": {"value": "<9000 spaces>", "revision": r}}` answers `400 INVALID_SESSION` "The pasted text is too long", because `leboncoin.ParseSessionInput` checks `len(raw) > 8192` before trimming. FR-SECRET-014 requires such an input to be untouched.

### Backend (TS-SECRET-017, TS-SECRET-018)

`internal/service/settings.go`, `sessionChangeFor`, after the `Remove` branch and before `ParseSessionInput`:

```go
if strings.TrimSpace(input.Value) == "" {
	return nil, nil // whitespace only, any length: untouched, no revision check (FR-SECRET-014)
}
```

- The function's doc comment becomes "…or nil when the session is untouched (no input, or a whitespace-only value of any length)."
- Rule order for a `value`: whitespace only → untouched; otherwise `ParseSessionInput` unchanged (raw length check first, then trimming, prefixes, extraction, validation). So `W9000` + `abc` stays `INVALID_SESSION` "too long" and is never trimmed into `abc`; `W100` + `datadome=N` + `W100` stays a replacement.
- The rule uses `strings.TrimSpace`, the trimming `ParseSessionInput` and `tokenChangeFor` already apply to short input, so "blank" means the same at every length (spaces, tabs, `\n`, `\r` and other Unicode white space, like the frontend's `trim()`).
- `ParseSessionInput` keeps its behaviour and tests. Only its stale doc comment is corrected to "An empty result without error means the input was blank." (the "session must be cleared" meaning was dropped by issue #265).
- An untouched session gives `sessionChange = nil`, so `store.SaveSettings` only reads the session: value, `revision`, `updatedAt`, status, expiry, revocation and `lastAttempt` are kept, no revision check runs, and a concurrent renewal is kept (FR-SECRET-008). A token part in the same request is processed as before.
- Claude token (TS-SECRET-018): `tokenChangeFor` already trims before `validateClaudeToken`; no change.
- Request size limit unchanged: a body over 32 768 bytes is still `413 REQUEST_TOO_LARGE` in the handler, before the service runs; nothing is saved.
- No persistence, migration, concurrency or logging change.

### API

Compatible behaviour change: a whitespace-only session `value` longer than 8192 characters, formerly `400 INVALID_SESSION` "too long", is now `200` and untouched. `API_SPECIFICATION.md` ("LeBoncoin session input", "Save settings") says that a whitespace-only value is untouched whatever its length and that the 8192 / 1024-character limits apply only to input that is not whitespace only. No field, status code or message changes.

### Frontend (TS-SECRET-019)

Not affected: `SettingsModal` sends a part only when `value.trim()` / `claudeValue.trim()` is non-empty, has no length check of its own, and closes without a request when both entries are untouched. A long blank paste therefore sends no part and shows no message. TC-005 adds e2e tests only.

### Tests (issue #286)

Test names start with `TestLongWhitespace_TC00n_` (Go) or `long whitespace TC-00n` (Playwright), so they do not collide with the #265 `TC-` numbers.

| Test case | Test file(s) |
| --- | --- |
| TC-001 | `internal/service/long_whitespace_test.go` (new, `package service`, table test calling `sessionChangeFor` directly) |
| TC-002 | `internal/httpapi/secret_masking_test.go` (existing helpers `withSecretNet`, `storeSecrets`, `finishSessionAttempt`, `rejectStoredToken`, `assertStoredSession`, `assertStoredToken`, `assertLastCookie`, `assertNoSecret`; a 40 000-space value for `413`) |
| TC-003 | `internal/httpapi/secret_masking_test.go` (`secretNet.setClaude`, `verifications`) |
| TC-004 | `internal/httpapi/secret_masking_test.go` (renewal to Y through `store.FinishLeboncoinSessionAttempt`) |
| TC-005 | `tests/leboncoin-session-settings.spec.ts` (session cases, both blank, too long non-blank, Cancel); `tests/claude-token-settings.spec.ts` (`C1025` alone, `W9000` with U) |

The existing check `internal/httpapi/settings_test.go:127` (8193 `v` → too long) stays valid. Commands: `go test ./...`, `go vet ./...`, `npm run build`, `npx playwright test`.

## Files to change

- Backend: `internal/model/model.go`, `internal/store/store.go`, `internal/store/claude_token.go`, `internal/service/settings.go`, `internal/service/service.go`, `internal/httpapi/settings.go`, `internal/httpapi/server.go`.
- Frontend: `src/api/settings.ts`, `src/components/SettingsModal.tsx`, `src/index.css` (only if needed).
- Documentation: `doc/leboncoin-session.md` (done in this stage), this file, and the amendment notes in the [LeBoncoin session settings](../leboncoin-session-settings/technical.md), [Claude token settings](../claude-token-settings/technical.md) and [session capture](../leboncoin-session-capture/technical.md) technical specifications.
- No change to the schema, the collector, AI reviews, auth, `playwright.config.ts`, `package.json` or `scripts/`.
- Issue #286: `internal/service/settings.go` (rule and doc comment), `internal/leboncoin/session.go` (doc comment only), tests listed in "Tests (issue #286)"; `API_SPECIFICATION.md` and the TS-LBC-SET-009 / TS-CLT-SET-011 rows are updated in this technical stage. No frontend, schema or refactoring change; no manual follow-up.

## Tests

New Playwright test files would need `playwright.config.ts` `testMatch`, which the developer cannot edit, so the e2e cases go into the existing spec files.

| Test case | Test file(s) |
| --- | --- |
| TC-001 | `internal/httpapi/secret_masking_test.go` |
| TC-002 | `internal/httpapi/secret_masking_test.go` (all routes, error paths, captured `log` output) |
| TC-003 | `internal/httpapi/secret_masking_test.go` |
| TC-004 | `internal/httpapi/secret_masking_test.go` |
| TC-005 | `internal/httpapi/secret_masking_test.go` |
| TC-006 | `internal/service/secret_masking_test.go` (fake Claude) and `internal/httpapi/secret_masking_test.go` |
| TC-007 | `internal/httpapi/secret_masking_test.go` |
| TC-008 | `internal/httpapi/secret_masking_test.go` |
| TC-009 | `internal/httpapi/secret_masking_test.go` (`store.FinishLeboncoinSessionAttempt` for renewal and revocation, stored `expires_at` for expiry) |
| TC-010 | `internal/httpapi/secret_masking_test.go` (rows written by `store.SaveLeboncoinSession` / `store.SaveSettings`, store closed and reopened on the same data directory) |
| TC-011 | `internal/model/secret_hint_test.go` |
| TC-012 | `tests/leboncoin-session-settings.spec.ts`; Claude helper corner in `tests/claude-token-settings.spec.ts` |
| TC-013 | `tests/leboncoin-session-settings.spec.ts` (session only, both empty, whitespace, errors, double Save); `tests/claude-token-settings.spec.ts` (token only, same token again, Claude errors) |
| TC-014 | `tests/claude-token-settings.spec.ts` (Remove token); `tests/leboncoin-session-settings.spec.ts` (Remove session, undo, cancel, combined save, kept on error) |
| TC-015 | `tests/leboncoin-session-settings.spec.ts` |
| TC-016 | `tests/leboncoin-session-settings.spec.ts` (session warnings); `tests/claude-token-settings.spec.ts` (token warning) |
| TC-017 | `tests/leboncoin-session-settings.spec.ts` |
| TC-018 | `tests/leboncoin-session-settings.spec.ts` |
| TC-019 | `internal/service/leboncoin_session_docs_test.go` |

Existing tests to adapt to the new contract (they assert the removed behaviour): `internal/httpapi/settings_test.go` (`value` fields, legacy `PUT` → `405`), `internal/httpapi/claude_settings_test.go` (`value` fields, `{}` → `200`, equal token now verified, empty values untouched), `internal/httpapi/ownership_test.go` (legacy `PUT` → `PUT /api/v1/settings`), `internal/service/session_test.go` (`SaveLeboncoinSession` helper → `SaveSettings`), `internal/service/settings_test.go` ("unchanged token re-verified" → always verified), `tests/leboncoin-session-settings.spec.ts` and `tests/claude-token-settings.spec.ts` (full value in inputs, empty save clears), and the session mock in `tests/responsive-layout.spec.ts` (`value` → `saved`/`hint`).

Commands: `go test ./...`, `go vet ./...`, `npm run build`, `npx playwright test`.

## Refactoring

- Removal of `PUT /api/v1/settings/leboncoin-session` and `Service.SaveLeboncoinSession`. Reason: the route returns the full value and clears the session on an empty value, which contradicts FR-SECRET-001/006; the frontend no longer calls it. Scope: the `PUT` branch in `internal/httpapi/server.go`, the service method, and the tests that used them. Risk: low; no other client exists, and the store method and its tests are unchanged.
- No other refactoring.

## Risks and follow-ups

- A client that sends a bare 4-character hint as a value replaces the secret with it; only the frontend's rule prevents this (covered by TC-013). Accepted: the server cannot distinguish a hint from a short valid value.
- Manual follow-ups: none.

## Unresolved questions

None.

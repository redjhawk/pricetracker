# Technical specification: LeBoncoin session settings

Status: ready (API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized)\
Functional specification: [functional.md](functional.md), FR-LBC-SET-001 – 019 (ready, Q-SET-1 resolved 2026-10-03)\
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section “LeBoncoin session settings (approved 2026-10-03)”\
Companions: [header menu](../app-header-menu/technical.md), [session collection rev. 2](../leboncoin-session-collection/technical.md), [file removal](../leboncoin-session-file-removal/technical.md)

## Requirement mapping

`SET-*` abbreviates `FR-LBC-SET-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-LBC-SET-001 | SET-012, SET-016, SET-017 | Single-row SQLite table `leboncoin_session` with value, expiry, revocation, `revision` token and last session-attempt outcome; additive idempotent migration | `internal/store/store.go`, `internal/model/model.go` | Store tests: fresh DB, existing DB upgrade, reopen persistence, conditional update |
| TS-LBC-SET-002 | SET-005, SET-006, SET-008 | Server-side input parser `leboncoin.ParseSessionInput` (raw value or cookie string; trimming; prefixes; validation; size limits) | `internal/leboncoin/session.go` | Table-driven Go tests for every accepted/rejected example in the functional spec |
| TS-LBC-SET-003 | SET-002, SET-004, SET-007, SET-008, SET-010, SET-013, SET-017 | Service methods `LeboncoinSession` / `SaveLeboncoinSession` with optimistic `revision` check; save never triggers a check | `internal/service/service.go` | Service tests: save, clear, no-op clear, stale revision → conflict, no collection queued |
| TS-LBC-SET-004 | SET-002–010, SET-017, SET-018 | `GET` and `PUT /api/v1/settings/leboncoin-session` | `internal/httpapi/server.go` | HTTP handler tests: 200/400/405/409/413/500 bodies, `Cache-Control: no-store` |
| TS-LBC-SET-005 | SET-014, SET-015 | Response `status` (`none`/`active`/`expired`/`revoked`) and `lastAttempt` (`accepted`/`rejected`/`failed`) derived from the row; hint scoped to the revision that made the attempt | `internal/store/store.go`, `internal/service/service.go` | Store/service tests with injected times and outcomes |
| TS-LBC-SET-006 | SET-001–011, SET-014, SET-015, SET-017, SET-019 | Carbon `SettingsModal` with one `TextArea`, loading/error/saving/validation/conflict/hint states | `src/components/SettingsModal.tsx`, `src/api/settings.ts`, `src/api/client.ts` (R-1), `src/App.tsx`, `src/index.css` (only if layout needs it) | Playwright mocked-API spec `tests/leboncoin-session-settings.spec.ts`; manual keyboard/400 px QA |
| TS-LBC-SET-007 | SET-018 | Value only in this endpoint; never logged; no browser storage; API already `no-store` | handlers, service, collector logging | Log/response grep in Go tests and QA |

## Frontend

Follows the existing structure (`src/components`, `src/api`), not the generic `features/` layout of the skill, because the repository already established this convention.

**API module — `src/api/settings.ts`.** Exports:

```ts
export interface LeboncoinSessionSettings {
  value: string | null;
  revision: number;
  updatedAt: string | null;
  status: "none" | "active" | "expired" | "revoked";
  expiresAt: string | null;
  revokedAt: string | null;
  lastAttempt: { outcome: "accepted" | "rejected" | "failed"; attemptedAt: string } | null;
}
export function getLeboncoinSession(): Promise<LeboncoinSessionSettings>;
export function saveLeboncoinSession(value: string, revision: number): Promise<LeboncoinSessionSettings>;
```

Both unwrap the `{ "session": … }` envelope. They reuse the shared `request` helper and `ApiError` from `src/api/client.ts` (R-1, performed first). No duplicate client-side cookie parser: the server is authoritative for FR-LBC-SET-005/006, and its `400 INVALID_SESSION` message is shown on the field. The client sends the entry text exactly as typed (the server trims).

**Component — `src/components/SettingsModal.tsx`.** Props: `open`, `onClose`, `launcherButtonRef` (the header menu trigger, for focus return, FR-APP-MENU-004). Carbon `Modal` with `modalHeading="Settings"`, `primaryButtonText="Save"` (“Saving…” while pending), `secondaryButtonText="Cancel"`, `size="sm"`, `launcherButtonRef`. The single entry is a Carbon `TextArea` (id `leboncoin-session`, `labelText="LeBonCoin session"`, about 4 rows) because real values are long and must wrap without horizontal page scrolling (SET-019). Helper text: “Paste the datadome value or a cookie string containing datadome=…. Save an empty field to remove the session.” (SET-001).

State (local to the component; page-level state in `App.tsx` only holds `settingsOpen`):

| State | Trigger | Rendering / behavior |
| --- | --- | --- |
| `loading` | Each time `open` becomes true (SET-002) | `InlineLoading` “Loading settings…”; TextArea disabled; Save disabled. A request token/`active` flag discards a late response after close (same pattern as `App.tsx` detail loading). |
| `loadError` | GET fails (SET-003) | `InlineNotification kind="error"` “Could not load settings” with the safe message; TextArea not rendered; Save disabled; Cancel/close enabled. Reopening retries. |
| `loaded` | GET succeeds | TextArea = `value ?? ""` (SET-004), unmasked. Keep `revision` from the response. Show at most one hint (below). |
| `saving` | Save pressed (SET-009) | Save disabled and labelled “Saving…”, TextArea disabled, `onRequestClose` ignored (Escape/close/Cancel do nothing), a guard flag prevents a second request. |
| `invalid` | PUT → 400 `INVALID_SESSION` (SET-006) | TextArea `invalid` with `invalidText` = server message; input kept; modal open. Editing clears the error. |
| `saveError` | PUT → network error, 5xx or other code (SET-010) | `InlineNotification kind="error"` “Could not save settings”; input kept; Save re-enabled for retry. |
| `conflict` | PUT → 409 `SESSION_CHANGED` (SET-017) | `InlineNotification kind="warning"` (not dismissible) titled “Session changed” with text “The LeBoncoin session changed after Settings was opened. Nothing was saved. Copy your text if needed, then close and reopen Settings before saving.” Input kept, TextArea stays editable for copying, Save disabled until the modal is reopened. |
| success | PUT → 200 (SET-007, SET-008) | Call `onClose()`; the next open reloads from the server. Nothing is cached in React state after close, and nothing is written to `localStorage`/`sessionStorage` (SET-018). |

Cancel / Escape / close button (when not saving) discard edits (SET-011): on close the component resets its state so the next open shows the server value.

Hints (SET-014, SET-015), rendered as `InlineNotification kind="warning" lowContrast hideCloseButton` above the TextArea, in text (not colour only, SET-019); dates use the same `Intl.DateTimeFormat("en-GB", { day, month: "short", year, hour, minute, timeZone: "UTC", timeZoneName: "short" })` as `ItemDetail.tsx`:

- `status === "revoked"`: “LeBoncoin revoked this session on {revokedAt}. It is no longer used. Capture a new session and save it here, or save an empty field to stop using it.”
- `status === "expired"`: “This session expired on {expiresAt}. It is no longer used. Capture a new session and save it here, or save an empty field to stop using it.”
- otherwise, `status === "active"` and `lastAttempt.outcome === "rejected"`: “LeBoncoin rejected this session on {attemptedAt}. Capture a new session and save it here.”
- otherwise, `status === "active"` and `lastAttempt.outcome === "failed"`: “The last LeBoncoin check using this session failed on {attemptedAt} for a reason other than a rejection.”
- No hint for `none`, for `lastAttempt === null` (unused value) or for `accepted`.

Accessibility: Carbon `Modal` provides focus trap, initial focus and Escape; `launcherButtonRef` returns focus to the header control. Errors use the TextArea `invalidText` (associated with the field) or `InlineNotification` (role `status`/`alert` from Carbon). At about 400 px the `sm` modal becomes full width and the TextArea wraps.

`App.tsx`: add `settingsOpen` state and a `menuButtonRef`; render `<AppMenu>` (header subject) and `<SettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} launcherButtonRef={menuButtonRef} />`. No route/URL is added. No change to item state or polling.

## Backend

### Persistence (TS-LBC-SET-001)

Additive migration inside `Store.createSchema` (runs on every start, idempotent; existing tables and data untouched):

```sql
CREATE TABLE IF NOT EXISTS leboncoin_session (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  value TEXT CHECK (value IS NULL OR length(value) BETWEEN 1 AND 4096),
  expires_at TEXT,
  revoked_at TEXT,
  revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
  updated_at TEXT,
  last_attempt_at TEXT,
  last_attempt_outcome TEXT CHECK (last_attempt_outcome IS NULL OR last_attempt_outcome IN ('accepted', 'rejected', 'failed'))
);
INSERT OR IGNORE INTO leboncoin_session (id, revision) VALUES (1, 0);
```

- One row (`id = 1`) because the application has one global setting (no accounts). `value IS NULL` means no session (sessionless collection).
- `expires_at`: known expiry (UTC, `timestampLayout`) supplied by LeBoncoin in a verified renewal; `NULL` for an operator-pasted value (FR-LBC-COL-015: a pasted value has no known expiry).
- `revoked_at`: set when LeBoncoin explicitly deleted the cookie; the value is kept so the modal can show it with the warning (SET-015).
- `revision`: optimistic concurrency token. Incremented on every change of the stored session that the modal must not silently overwrite: operator save/clear, renewal that changes the value, revocation. Recording an attempt outcome alone, or an expiry-only renewal with the same value, does not increment it.
- `last_attempt_at` / `last_attempt_outcome`: latest session-assisted attempt for the current revision lineage (TS-LBC-SET-005). Cleared by every operator save/clear.
- Downgrade: an older binary ignores the table. No data from the revision 1 file/sidecar is imported (FR-LBC-RM-002).

Model (`internal/model/model.go`), API-shaped so handlers stay thin:

```go
type LeboncoinSession struct {
	Value       *string                   `json:"value"`
	Revision    int64                     `json:"revision"`
	UpdatedAt   *time.Time                `json:"updatedAt"`
	Status      string                    `json:"status"`
	ExpiresAt   *time.Time                `json:"expiresAt"`
	RevokedAt   *time.Time                `json:"revokedAt"`
	LastAttempt *LeboncoinSessionAttempt  `json:"lastAttempt"`
}
type LeboncoinSessionAttempt struct {
	Outcome     string    `json:"outcome"`
	AttemptedAt time.Time `json:"attemptedAt"`
}
```

Store methods (parameterized SQL, no network I/O in transactions):

- `LeboncoinSession(ctx) (model.LeboncoinSession, error)` reads the row; `Status` is derived with the read time: `none` if value is NULL; `revoked` if `revoked_at` is set; `expired` if `expires_at <= now`; else `active`.
- `SaveLeboncoinSession(ctx, value *string, expectedRevision int64, now time.Time) (model.LeboncoinSession, error)`; in one transaction: read the row; if `revision != expectedRevision` return `store.ErrSessionChanged`; if `value == nil` and the stored value is already NULL return the current row unchanged (SET-008 “changes nothing”); otherwise `UPDATE … SET value = ?, expires_at = NULL, revoked_at = NULL, last_attempt_at = NULL, last_attempt_outcome = NULL, revision = revision + 1, updated_at = ? WHERE id = 1 AND revision = ?` and return the re-read row. Saving the same value again is a normal save (increments the revision and clears the hint, SET-007).
- `FinishLeboncoinSessionAttempt(ctx, startRevision int64, outcome, now)`: specified in the [collection design](../leboncoin-session-collection/technical.md) (TS-LBC-COL-008).

### Input parsing (TS-LBC-SET-002)

`leboncoin.ParseSessionInput(raw string) (value string, err error)` in `internal/leboncoin/session.go` (the LeBoncoin package owns the `datadome` cookie rules). `value == ""` with `err == nil` means “clear”. Errors are a `*leboncoin.SessionInputError` whose `Message` is safe to display. Rules, in order:

1. If `len(raw) > 8192` bytes: “The pasted text is too long (maximum 8192 characters).”
2. `s := strings.TrimSpace(raw)`; empty → clear (SET-008).
3. Strip an optional case-insensitive prefix `Cookie:` or `Set-Cookie:` and trim again. If a prefix was stripped and the remainder is empty → “No datadome cookie was found in the pasted text.” (e.g. `Cookie:`, `set-cookie:  `; `Cookie: ;` reaches rule 5 with the same message).
4. If `s` contains neither `=` nor `;` it is a raw value; validate it (rule 6).
5. Otherwise it is a cookie string: split on `;`; trim each part; skip empty parts; a part without `=` (e.g. `Secure`, `HttpOnly`) is an attribute and ignored; for `name=value` split at the first `=`, trim name and value; collect values whose name is exactly `datadome` (cookie names are case-sensitive). Then:
   - none → “No datadome cookie was found in the pasted text.”
   - any collected value empty → “The datadome cookie value is empty.”
   - two or more distinct values → “The pasted text contains different datadome values; paste only one.”
   - identical duplicates → one value.
   All other pairs/attributes are discarded and never stored (SET-005).
6. Value validation (same rule as the existing `validCookie`): 1 to 4096 bytes, each byte in 0x21–0x7E and not `"`, `,`, `;` or `\`. Failure: “The datadome value contains spaces, quotes or other characters that are not allowed in a cookie value.” or, for length, “The datadome value is too long (maximum 4096 characters).”

Consequences (documented in tests): `abc123` → `abc123`; `datadome=abc123` → `abc123`; `Cookie: a=1; datadome=abc123; b=2` → `abc123`; `datadome=abc123; Max-Age=31536000; Domain=.leboncoin.fr; Path=/; Secure` → `abc123`; a value with internal whitespace or line break → invalid; a raw value containing `=` is interpreted as a cookie string (real `datadome` values do not contain `=`). The value displayed by the modal is always re-accepted unchanged (SET-004).

### Service (TS-LBC-SET-003, TS-LBC-SET-005)

In `internal/service/service.go`:

- `LeboncoinSession(ctx) (model.LeboncoinSession, error)` → store read.
- `SaveLeboncoinSession(ctx, raw string, revision int64) (model.LeboncoinSession, error)`: parse; on `SessionInputError` return `&Error{400, "INVALID_SESSION", msg}`; call the store with `nil` for clear; map `store.ErrSessionChanged` to `&Error{409, "SESSION_CHANGED", "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving."}`. It does not queue any collection and does not touch items (SET-013). Because collection reads the store at the start of each attempt (TS-LBC-COL-007), the new state applies to the next attempt without restart.
- Ordering with in-flight attempts: an attempt captures `revision` at its start; its completion write is conditional on that revision (TS-LBC-COL-008), so an operator save always wins over an older attempt (FR-LBC-COL-013, SET-017).

### HTTP (TS-LBC-SET-004)

In `internal/httpapi/server.go`, before the item routes: path `/api/v1/settings/leboncoin-session`.

- `GET` → `200 {"session": <LeboncoinSession>}`; store error → existing `serverError` (500 `INTERNAL_ERROR`).
- `PUT` → body limited by `http.MaxBytesReader` to 32 768 bytes (as `POST /items`); decode into `struct { Value *string `json:"value"`; Revision *int64 `json:"revision"` }`; reject trailing JSON as `POST /items` does. Missing `value` or `revision`, a JSON type mismatch (`*json.UnmarshalTypeError`, e.g. a non-string `value` or non-integer `revision`), or `revision < 0` → `400 INVALID_REQUEST` “Request must include a session value (use an empty string to remove it) and the revision from Settings.” Over-size → `413 REQUEST_TOO_LARGE`; bad JSON → `400 INVALID_JSON`. Then call the service; success → `200 {"session": …}`; typed errors via existing `serviceError`.
- Other methods → `405 METHOD_NOT_ALLOWED` with `Allow: GET, PUT`.
- `Cache-Control: no-store` is already set for all `/api/` responses. Do not log request bodies; `serverError` logs only store errors, which never contain the bound value (parameterized SQL).

## API

Two new endpoints, specified in the pending proposal of [API_SPECIFICATION.md](../../../API_SPECIFICATION.md). Existing item endpoints, shapes and statuses are unchanged; `lastAttempt.message` texts for LeBoncoin session-assisted failures change wording (free text, already displayed as-is). This document does not constitute approval; the frontend and backend must not be implemented before the user confirms the contract.

## Scope and refactoring

In scope: the files listed in the mapping, plus tests. Out of scope: other settings, masking, a Clear button, triggering checks on save, any change to items responses.

R-1 approved by the user ("Do it first") and performed before the feature; `request`/`ApiError` live in `src/api/client.ts`. No other refactoring.

## Verification and unresolved questions

- Go (`go test ./...`, `go test -race ./...`): parser table (all SET-005/006 examples, 8192/4096 limits, prefixes, duplicates); store migration on a fresh DB and on a DB created by the previous schema (copy of a fixture created with current `createSchema`), reopen persistence (SET-012); save/clear/no-op clear/stale revision; status derivation (none/active/expired/revoked) with injected `now`; handler tests via `httptest` for every status code and body, `Allow` header, `no-store`, and that the value does not appear in log output captured with `log.SetOutput`.
- Frontend: `npm run build` (type check). New Playwright spec `tests/leboncoin-session-settings.spec.ts` with a mocked API (same approach as `tests/platform-tabs.spec.ts`; add it to `testMatch` in `playwright.config.ts` and an npm script `test:settings`): open via menu; loading; load error disables save; full value shown; save success closes and reopens with the new value; empty save; 400 shows field error; 500 keeps input; 409 shows reload warning and disables save; Escape/cancel discards; save in progress blocks close and double submit (assert one PUT); hints for rejected/failed/expired/revoked/accepted/null; focus returns to the menu button; 400 px viewport has no horizontal scroll; no `localStorage`/`sessionStorage` entries.
- QA (later stage): real application with an isolated data directory; set/clear/conflict from two browser contexts; restart persistence.

Clarification for the user (non-blocking, proposed default): FR-LBC-SET-017 lists “expiry” among changes that make a save stale. Natural expiry is time passing, not a stored change, so it does not increment `revision`; a modal opened before the expiry time can still save successfully (the operator is replacing or clearing the value anyway). Revocation by LeBoncoin and every renewal of the value do increment it. If the user wants natural expiry to also force a reload, the server would need to compare the expired state at load and save time; this is not proposed.

## Revision note

2026-10-03, review amendments: REV-SET-001: status updated to record the API approval of 2026-10-03; R-1 performed first (`src/api/client.ts`). REV-SET-003: TS-LBC-SET-002 rule 3 clarified for an empty remainder after a header prefix.

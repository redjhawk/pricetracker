# Technical specification: Claude token settings

Status: ready
Functional specification: [functional.md](functional.md), FR-CLT-SET-001–011 (ready, 2026-10-04)
API: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md), section "Claude token settings and AI reviews (2026-10-04)"
Companions: [LeBoncoin AI review technical](../leboncoin-ai-review/technical.md), [LeBoncoin session settings technical](../leboncoin-session-settings/technical.md)

## Requirement mapping

`CLT-*` abbreviates `FR-CLT-SET-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-CLT-SET-001 | CLT-007, CLT-009 | Single-row SQLite table `claude_token` (value, revision, updated_at, last_rejected_at); additive idempotent migration | `internal/store/store.go`, `internal/model/model.go` | Store tests: fresh DB, upgrade of existing DB, reopen persistence |
| TS-CLT-SET-002 | CLT-004, CLT-011 | `internal/claude` package: plain `net/http` Messages API client with `Verify(ctx, token)` (1-token call) and typed errors | `internal/claude/client.go` | `httptest` server tests: 200 → ok, 401/403 → rejected, 5xx/429/network → unreachable; headers asserted |
| TS-CLT-SET-003 | CLT-004, CLT-005, CLT-006, CLT-011 | Service `ClaudeToken` and atomic `SaveSettings` (both entries in one validation + one transaction) | `internal/service/service.go`, `internal/store/store.go` | Service tests with fake verifier: invalid Claude + valid LeBoncoin saves nothing; unchanged token not re-verified; clear without verification |
| TS-CLT-SET-004 | CLT-002–006, CLT-008 | `GET /api/v1/settings/claude-token`, `PUT /api/v1/settings` | `internal/httpapi/server.go` | Handler tests for every status/code in the API table, `Cache-Control: no-store` |
| TS-CLT-SET-005 | CLT-009 | `lastRejectedAt` set by reviews on 401/403/429 conditionally on the token revision; cleared on every save | store, service (review worker, see TS-LBC-AIR-006) | Store test: rejection with stale revision ignored; save clears it |
| TS-CLT-SET-006 | CLT-001–003, CLT-006, CLT-009–011 | Second Carbon `TextInput` entry in `SettingsModal`; save through `PUT /api/v1/settings` | `src/components/SettingsModal.tsx`, `src/api/settings.ts` | Playwright mocked-API spec `tests/claude-token-settings.spec.ts`; manual keyboard/400 px QA |
| TS-CLT-SET-007 | CLT-008 | Token only in the settings endpoints and the outgoing `Authorization` header; never logged, never in error messages, items or reviews | `internal/claude`, service, handlers | Go tests grep logs/responses for the token; QA |
| TS-CLT-SET-008 | CLT-012 (issue #56) | Helper text of the Claude token entry states the token is used for Amazon and LeBoncoin AI reviews; exact text in [Amazon AI review technical](../amazon-ai-review/technical.md). Frontend text only; backend and API not affected | `src/components/SettingsModal.tsx` | Playwright text assertion |

## Frontend

Existing structure is kept (`src/components`, `src/api`).

**API module — `src/api/settings.ts`** (additions; existing exports unchanged and still used for loading):

```ts
export interface ClaudeTokenSettings {
  value: string | null;
  updatedAt: string | null;
  lastRejectedAt: string | null;
}
export function getClaudeToken(): Promise<ClaudeTokenSettings>; // unwraps { claudeToken }
export function saveSettings(input: {
  leboncoinSession: { value: string; revision: number };
  claudeToken?: { value: string };
}): Promise<{ session: LeboncoinSessionSettings; claudeToken: ClaudeTokenSettings }>;
```

`saveLeboncoinSession` stays exported but the modal no longer calls it (the endpoint remains in the contract).

**Component — `SettingsModal.tsx`.**

- Load: on open, `Promise.all([getLeboncoinSession(), getClaudeToken()])`. Loading/load-error states are shared: while loading or after a failure both entries are disabled/not rendered and Save is disabled (CLT-002). One load error notification covers both.
- New entry below the LeBoncoin `TextArea`: Carbon `TextInput` (`id="claude-token"`, `labelText="Claude token"`, `type="text"`, unmasked per D-2, `autoComplete="off"`, `spellCheck={false}`). Helper text: "Paste the token printed by `claude setup-token` (Claude Pro/Max subscription). Save an empty field to remove it." When the loaded value is `null`, append "AI reviews are unavailable until a token is saved." (CLT-001, CLT-003). A single-line input is enough: tokens have no whitespace; at 400 px the `sm` modal is full-width and the input scrolls internally, not the page (CLT-010).
- Warning (CLT-009): if `lastRejectedAt` is not null, an `InlineNotification kind="warning" lowContrast hideCloseButton` directly above the Claude entry: "Claude rejected this token for an AI review on {date}. Replace it with a new token from `claude setup-token`." Date format: the modal's existing `formatDate`.
- Save (CLT-004/005/006): always send `leboncoinSession` (preserves the current save behavior of FR-LBC-SET-007); add `claudeToken: { value }` only when the trimmed entry differs from the loaded value (`loaded.value ?? ""`), so an unchanged token is never re-verified. While saving: Save shows "Saving…", both entries disabled, close ignored (existing guard), `InlineLoading` "Saving settings…" (replaces "Saving the session…"). No client-side token validation: the server is authoritative.
- Error mapping (by `ApiError.code`):

| Code | Rendering |
| --- | --- |
| `INVALID_SESSION`, `SESSION_CHANGED` | Existing LeBoncoin behavior, unchanged |
| `INVALID_CLAUDE_TOKEN`, `CLAUDE_TOKEN_REJECTED`, `CLAUDE_UNREACHABLE` | `invalid` + `invalidText` = server message on the Claude `TextInput` (associated with the entry, announced). Editing the entry clears it. Input kept, modal open. |
| other / network | Existing "Could not save settings" notification |

- Success closes the modal; nothing is cached after close and nothing is written to browser storage (CLT-008). Reopening reloads both values.

## Backend

### Persistence (TS-CLT-SET-001)

Added to `createSchema` (idempotent; existing data untouched):

```sql
CREATE TABLE IF NOT EXISTS claude_token (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  value TEXT CHECK (value IS NULL OR length(value) BETWEEN 1 AND 1024),
  revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
  updated_at TEXT,
  last_rejected_at TEXT
);
INSERT OR IGNORE INTO claude_token (id, revision) VALUES (1, 0);
```

Model (`internal/model/model.go`):

```go
type ClaudeToken struct {
	Value          *string    `json:"value"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	LastRejectedAt *time.Time `json:"lastRejectedAt"`
	Revision       int64      `json:"-"` // internal: used by reviews (TS-CLT-SET-005)
}
```

Store methods:

- `ClaudeToken(ctx) (model.ClaudeToken, error)`.
- `SaveSettings(ctx, session *SessionChange, token *TokenChange, now) (model.LeboncoinSession, model.ClaudeToken, error)` — one transaction. `SessionChange{Value *string; ExpectedRevision int64}` applies exactly the existing `SaveLeboncoinSession` rules (revision check → `ErrSessionChanged`, no-op clear, update). `TokenChange{Value *string}` sets `value`, `revision = revision + 1`, `updated_at = now`, `last_rejected_at = NULL`; clearing an already-empty token changes nothing. Either part may be nil.
- `MarkClaudeTokenRejected(ctx, revision, now)`: `UPDATE claude_token SET last_rejected_at = ? WHERE id = 1 AND revision = ?`.
- `ClearClaudeTokenRejected(ctx, revision)`: same condition, sets `NULL` (called after a successful review, so the warning reflects the *latest* attempt).

**Refactoring R-CLT-1 (performed with the feature):** extract the body of `Store.SaveLeboncoinSession` into an unexported `saveLeboncoinSessionTx(ctx, tx, …)` used by both `SaveLeboncoinSession` and `SaveSettings`. Reason: the atomic two-entry save (CLT-006) must reuse the same session rules inside one transaction without duplicating them. Scope: `internal/store/store.go` only; behavior of the existing method unchanged (its tests must still pass). Risk: low. Decision: perform, in PR 2.

### Claude client (TS-CLT-SET-002)

New package `internal/claude` (no new dependency):

```go
const (
	Model         = "claude-sonnet-5-5"
	apiURL        = "https://api.anthropic.com/v1/messages"
	systemPreamble = "You are Claude Code, Anthropic's official CLI for Claude."
)
var (
	ErrRejected    = errors.New("claude rejected the token")     // 401, 403
	ErrUsageLimit  = errors.New("claude usage limit reached")    // 429
	ErrUnreachable = errors.New("claude could not be reached")   // network, timeout, 5xx, other non-2xx
	ErrBadResponse = errors.New("claude returned an unusable response")
)
type Client struct { http *http.Client; url string } // url overridable in package tests
func NewClient() *Client // http.Client{Timeout: 120 * time.Second}
func (c *Client) Verify(ctx context.Context, token string) error
```

Every request: `POST`, headers `Authorization: Bearer <token>`, `anthropic-version: 2023-06-01`, `anthropic-beta: oauth-2025-04-20`, `content-type: application/json`. The `system` field is an array whose **first block is exactly** `systemPreamble` (required for subscription OAuth tokens); further blocks follow. `Verify` sends `model: Model`, `max_tokens: 1`, system `[preamble]`, one user message `"ping"`, with a 20 s context timeout. 2xx → nil. Status mapping above; the response body is read (limited to 1 MiB) only to parse success responses and is never logged or returned. Logs record only the operation and HTTP status.

### Service (TS-CLT-SET-003)

- Service gets a `claude` field of interface type `claudeClient { Verify(ctx, token) error; Review(ctx, token, claude.ReviewInput) (model.AIReviewContent, error) }` so tests can fake it (same pattern as `leboncoinCollector`).
- `ClaudeToken(ctx)` → store.
- `SaveSettings(ctx, input SettingsInput) (model.LeboncoinSession, model.ClaudeToken, error)`, steps in order (first error wins, nothing saved):
  1. If `input.LeboncoinSession` present: `leboncoin.ParseSessionInput` → `400 INVALID_SESSION`.
  2. If `input.ClaudeToken` present: trim; empty → clear. Else reject (`400 INVALID_CLAUDE_TOKEN`) when longer than 1024 bytes ("The Claude token is too long (maximum 1024 characters).") or containing any byte ≤ 0x20 or 0x7F or non-ASCII ("The Claude token must not contain spaces, line breaks or other control characters.").
  3. If a non-empty token differs from the stored value: `Verify` outside any transaction. `ErrRejected` → `422 CLAUDE_TOKEN_REJECTED` "Claude refused this token. Create a new one with claude setup-token."; any other error (including 429) → `502 CLAUDE_UNREACHABLE` "The token could not be verified because Claude could not be reached. Try again." Equal to the stored value → no verification, no change.
  4. `store.SaveSettings` in one transaction; `ErrSessionChanged` → existing `409 SESSION_CHANGED`.
- Saving never starts a review or a price check (FR-LBC-AIR-008). Reviews read the token at their start (TS-LBC-AIR-006), so a new token applies without restart (CLT-004).

### HTTP (TS-CLT-SET-004)

In `handleAPI`, before item routes: `/api/v1/settings/claude-token` (GET only; `Allow: GET`) and `/api/v1/settings` (PUT only; `Allow: PUT`). The PUT decoder mirrors `handleLeboncoinSession`: 32 768-byte limit, single JSON value, `INVALID_REQUEST` for wrong types, missing both parts, `leboncoinSession` without string `value` or non-negative `revision`, or `claudeToken` without string `value`. The handler passes raw strings to the service; responses use `writeJSON`.

## API

Contract: [API_SPECIFICATION.md](../../../API_SPECIFICATION.md) "Claude token settings and AI reviews (2026-10-04)" — `GET /api/v1/settings/claude-token` and `PUT /api/v1/settings`. Additive: the existing LeBoncoin session endpoints and all item fields are unchanged. The token value appears only in these two responses (CLT-003, CLT-008).

## Scope and refactoring

Change boundary: `internal/claude/` (new), `internal/model/model.go`, `internal/store/store.go`, `internal/service/service.go`, `internal/httpapi/server.go`, `src/api/settings.ts`, `src/components/SettingsModal.tsx`, tests (`internal/**/_test.go`, `tests/claude-token-settings.spec.ts`). No config/env variable (CLT-007). Refactoring R-CLT-1 above (perform). No other refactoring.

Risks: the Anthropic OAuth beta header or the required preamble may change upstream; they are isolated as constants in `internal/claude`. A save with a slow Claude response waits up to 20 s; the UI shows progress.

## Verification and unresolved questions

- `go build ./...`, `go test ./...`, `npm run build`, `npx playwright test tests/claude-token-settings.spec.ts`.
- QA: open Settings (two entries, loading, load failure); save valid token (closes, reopen shows it); invalid token (error on entry, nothing saved including a changed LeBoncoin entry); unreachable (stop network / mock); whitespace inside token; clear; unchanged token save does not call Claude; rejected-token warning after a failed review, gone after new save; restart persistence; 400 px; keyboard; token absent from item responses and logs.
- Unresolved questions: none.

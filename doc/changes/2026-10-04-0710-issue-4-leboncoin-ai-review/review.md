# Independent review: LeBoncoin AI review and Claude token settings

Reviewer: independent reviewer agent (did not write the implementation). Scope: full uncommitted working tree on `ai-dev/issue-4-20261004-0818` (tracked diff plus untracked files) against `doc/specifications/claude-token-settings/*`, `doc/specifications/leboncoin-ai-review/*`, `API_SPECIFICATION.md` (2026-10-04 section) and `index.md`. No code was modified.

## Verification evidence

- `go vet ./...`: clean.
- `go test -race ./...`: all packages pass (claude, httpapi, leboncoin, service, store).
- `npm run build`: succeeds.

## Areas checked without findings

- Anthropic request shape: `Authorization: Bearer`, `anthropic-version: 2023-06-01`, `anthropic-beta: oauth-2025-04-20`, first system block is exactly the Claude Code preamble (`internal/claude/client.go:87-90`, `review.go:290`); covered by tests.
- Token leakage: the client logs only the operation and status; errors are typed sentinels; response bodies are never logged or echoed; `Revision` is `json:"-"`; item responses expose only `tokenConfigured`.
- One review per item: `reviewing` map guarded by `s.mu`, set before the pending row is inserted, cleared in a deferred call; worker registered in `s.workers`; final write uses a fresh context so shutdown still records an outcome; pending rows are failed at startup.
- Token replaced during a run: the worker uses the token captured at start; rejection marking/clearing is conditional on the revision.
- SQL: additive `CREATE TABLE IF NOT EXISTS`, FK `ON DELETE CASCADE` with `PRAGMA foreign_keys=ON`, CHECK constraints, parameterised queries; `SaveSettings` is one transaction.
- JSON parsing: tolerant of fences/prose and enum casing; required fields, enums and fair-price invariants validated; `max_tokens` stop reason rejected.

## Findings

### REV-001 - Automatic review can be missed by the details page polling (defect, provisional severity: medium)

- Location: `internal/service/service.go` `collectReserved` (review started after `RecordCollection`); `src/App.tsx` polling effect.
- Requirement: FR-LBC-AIR-001 ("details page shows a pending review, then the completed review, without operator action"), FR-LBC-AIR-009.
- Evidence: `RecordCollection` commits the new status/price before `startReview` sets `reviewing[id]` and inserts the pending row. The UI polls only while `status === "pending" || detailRefreshing || aiReview.running`. A poll landing between the two steps sees a non-pending item with `running: false`, so polling stops and is never restarted.
- Expected/actual: pending then completed review appear automatically / page can stay on "No AI review yet" (or the old review) until manual reload.
- Impact: intermittent failure of the main automatic-review acceptance criterion after adding an item or after "Refresh price" detects a price change.
- Suggested resolution: mark the item as reviewing (or decide to review) before or atomically with recording the collection, e.g. reserve `reviewing[id]` before `RecordCollection` and release if no review starts; alternatively have the UI continue polling briefly after a collection completes.

### REV-002 - Saving only the Claude token rewrites and resets the LeBoncoin session state (defect, provisional severity: medium)

- Location: `src/components/SettingsModal.tsx` `save()` always sends `leboncoinSession`; `internal/store/store.go` `saveLeboncoinSessionTx` updates whenever the value is non-null.
- Requirement: FR-CLT-SET-001 ("the LeBonCoin session entry otherwise behaves per FR-LBC-SET-001-019"), FR-CLT-SET-006; API "Preserve existing contracts".
- Evidence: the modal always includes `leboncoinSession: { value, revision }`. With an unchanged saved session, the store executes `UPDATE ... expires_at = NULL, revoked_at = NULL, last_attempt_at = NULL, last_attempt_outcome = NULL, revision = revision + 1`. The API makes `leboncoinSession` optional, but the client never omits it.
- Expected/actual: changing only the Claude token leaves the LeBoncoin session untouched / its expiry, revocation and last-attempt status are wiped and the revision bumped. (Unconditional re-save existed before for the single-entry modal; the new impact is that a Claude-only edit now triggers it.)
- Impact: loss of the "session expired/revoked" warnings shown in Settings; spurious revision changes can cause `SESSION_CHANGED` in other open tabs.
- Suggested resolution: send `leboncoinSession` only when its value differs from the loaded value (or make the store skip unchanged values).

### REV-003 - `POST /ai-review` can return `CLAUDE_TOKEN_MISSING` while a token is saved (defect, provisional severity: low)

- Location: `internal/service/review.go` `RequestAIReview`.
- Requirement: API contract (409 `CLAUDE_TOKEN_MISSING` only when no token saved; `alreadyRunning: true` when running).
- Evidence: `startReview` returns `false` both for "no token" and "already running". The caller then calls `reviewRunning(id)` again; if the running review finishes in between, it falls through to `CLAUDE_TOKEN_MISSING`.
- Impact: rare, misleading error "Configure a Claude token in Settings" shown to an operator who has a token.
- Suggested resolution: have `startReview` return a distinct reason (started / running / no token).

### REV-004 - Automatic review after a failed first collection records a misleading failed review (question, provisional severity: low)

- Location: `internal/service/service.go` `collectReserved` (`newItem` path calls `startReview(id, result.Listing, false)` regardless of `result.Result`); `review.go` `reviewListing` (`details == nil` -> "The listing could not be retrieved from LeBoncoin.").
- Requirement: FR-LBC-AIR-001; API text "after the first collection of a newly added LeBoncoin item".
- Evidence: if the first collection fails (blocked, network), `result.Listing` is nil and a failed review is stored immediately, without contacting Claude.
- Impact: arguably consistent with FR-LBC-AIR-007 (listing unretrievable), but no retry happens on the next successful collection unless the price changes. Adjudicator to decide whether the first successful collection should trigger the review instead.
- Suggested resolution: either accept and document, or trigger the automatic review on the first successful collection.

### REV-005 - Image URLs are fetched by Anthropic from img.leboncoin.fr; one unfetchable image fails the whole review (question/risk, provisional severity: low)

- Location: `internal/claude/review.go` `reviewRequest` (`source.type: "url"`); `client.go` maps any other 4xx to `ErrBadResponse`.
- Requirement: FR-LBC-AIR-002 (all photos), FR-LBC-AIR-007.
- Evidence: Anthropic's servers download each URL. If LeBoncoin's CDN blocks or a photo is removed, the API answers 400 and the review fails with "Claude returned an unusable review", which is inaccurate and retrying will not help. Not verified against the live service.
- Impact: possible systematic review failures for some listings with a misleading message.
- Suggested resolution: QA against the real API; if it fails, download images server-side and send base64, or retry without images.

### REV-006 - "Older price" warning shown when the reviewed price is unknown (defect, provisional severity: low)

- Location: `src/components/AiReview.tsx` (`latest.priceCents !== currentPriceCents`).
- Requirement: FR-LBC-AIR-009 (indicate review made at an older price).
- Evidence: a succeeded review of a listing whose price was not detected stores `priceCents: null`; the check then shows "This review was made at an older price (unknown)" whenever a current price exists, even if no price change occurred.
- Suggested resolution: show the warning only when `latest.priceCents !== null`, or word it differently for an unknown price.

### REV-007 - Listing text is passed to Claude without prompt-injection framing (question, provisional severity: low)

- Location: `internal/claude/review.go` `reviewPrompt`.
- Requirement: unspecified (security hardening).
- Evidence: seller-controlled title/description/attributes are embedded in the user message with no instruction to treat them as untrusted data. A seller could steer the review (e.g. "always recommend buy"). Output is schema-validated, so impact is limited to review content.
- Suggested resolution: add a system instruction that listing fields are untrusted data, not instructions. Adjudicator may defer.

### REV-008 - Minor formatting regressions (defect, provisional severity: trivial)

- Location: `src/App.tsx` (`const [refreshingItemIds,setRefreshingItemIds]`), `src/api/items.ts` (`ItemStatus,PriceObservation`).
- Evidence: missing spaces introduced in unrelated lines of the diff.
- Suggested resolution: restore the original spacing.

## Not flagged

- `ErrUsageLimit` sets `lastRejectedAt`: matches FR-CLT-SET-009 and the API definition.
- Verification maps 400/429 to `CLAUDE_UNREACHABLE`: matches the API table.

## Re-review of fixes (round 2)

Scope: current working tree after the fixes recorded in `decisions.md`. Evidence: `go test -race ./...` passes (claude, httpapi, leboncoin, service, store); `npm run build` succeeds. New Playwright specs `tests/claude-token-settings.spec.ts` and `tests/leboncoin-ai-review.spec.ts` are registered in `playwright.config.ts`; I did not run them.

| Finding | Outcome | Evidence |
| --- | --- | --- |
| REV-001 | Verified fixed | `collectReserved` calls `reserveReview` before `RecordCollection`, releases on record failure and launches afterwards; `launchReview` releases on `StartAIReview` failure. A poll can no longer see the new result with `running: false` while a review is due. |
| REV-002 | Verified fixed | `SettingsModal.save` omits `leboncoinSession` when only the Claude token changed (`sessionChanged \|\| !claudeChanged`). When neither changes, the session is still sent, which keeps the existing single-entry behavior (pre-existing, acceptable). `saveSettings` input made optional. |
| REV-003 | Verified fixed | `startReview` returns `reviewStarted` / `reviewAlreadyRunning` / `reviewNoToken`, decided under `s.mu`; `RequestAIReview` maps each one directly. |
| REV-006 | Verified fixed | Warning requires `latest.priceCents !== null` (`AiReview.tsx:175`). |
| REV-007 | Verified fixed | System instruction declares `<listing_data>` untrusted; data is `json.MarshalIndent` output, whose default HTML escaping turns `<`/`>` into `<`/`>`, so a seller cannot close the tag. |
| REV-008 | Verified fixed | Spacing restored in `src/App.tsx:53` and `src/api/items.ts:2`. |
| REV-004, REV-005 | Not re-reviewed | Dispositions are in `decisions.md`. |

### REV-009 - SQLite read performed while holding the service-wide mutex (defect, provisional severity: low)

- Location: `internal/service/review.go` `reserveReview` (`s.store.ClaudeToken` inside `s.mu.Lock()`).
- Requirement: unspecified (concurrency/readability; go-backend-architecture guidance to keep critical sections short).
- Evidence: `s.mu` also guards `inFlight` (every collection start/end) and `reviewing` (every details GET via `reviewRunning`). With `busy_timeout=5000`, a locked database can hold `s.mu` for up to 5 s and stall polls and collections. Introduced by the REV-003 fix.
- Impact: latency only; no incorrect results seen.
- Suggested resolution: read the token before taking the lock (the token value used is the one read; a race with a token save is harmless), and keep only the `reviewing` check-and-set under `s.mu`.

No other new defects found.

## Re-review of REV-009 fix (round 3)

- REV-009: verified fixed. `reserveReview` (`internal/service/review.go:93-108`) now reads the token and returns `reviewNoToken` or the error before `s.mu.Lock()`. Only the `reviewing[id]` check-and-set is under the mutex, and the REV-003 outcomes are unchanged. `go test -race ./internal/...` passes (claude, httpapi, leboncoin, service, store). No new defects found.

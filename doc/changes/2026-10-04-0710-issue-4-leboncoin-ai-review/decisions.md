# Review decisions: leboncoin-ai-review

Reviewed specification revisions: `doc/specifications/claude-token-settings/{functional,technical}.md`, `doc/specifications/leboncoin-ai-review/{functional,technical}.md`, `API_SPECIFICATION.md` (2026-10-04 section), working-tree snapshot on `ai-dev/issue-4-20261004-0818`.
Reviewed code revision: uncommitted working tree on top of `bfc3f24`.
Reviewer: independent reviewer agent (`review.md`).
Adjudicator: independent review adjudicator agent (did not write the code or the review).

## Findings and decisions

### REV-001: Automatic review can be missed by the details page polling

- Evidence: `internal/service/service.go:276-283` calls `RecordCollection` (commits non-pending status) before `startReview` marks `reviewing[id]` and inserts the pending row. `src/App.tsx` polls only while item pending, refreshing, or `aiReview.running`. A poll between the two steps stops polling permanently.
- Impact and scenario: operator adds a LeBoncoin item or refreshes a changed price; the details page can stay on "No AI review yet"/old review until manual reload.
- Criticality: critical, because it breaks the explicit acceptance criterion of FR-LBC-AIR-001 ("shows a pending review, then the completed review, without operator action") and FR-LBC-AIR-009 on the primary path; the race window is small but hit by every auto review and the consequence is a silent failure of the headline behavior.
- Disposition: fix.
- Reason: the specified behavior is unambiguous; the fix is local and server-side.
- Required change: make the review intent visible no later than the collection result. Reserve `reviewing[id]` (so `aiReview.running` is true) before `RecordCollection` when the item is LeBoncoin, a token is saved and (newItem or price changed); release it if no review starts. Equivalent atomic approaches are acceptable; a UI-only "poll a bit longer" heuristic is not sufficient alone.
- Specification decision: not applicable.
- Resolution: fixed and independently verified (review.md round 2). `collectReserved` calls `reserveReview` before `RecordCollection`, releases the reservation if recording fails, and launches the review afterwards. `launchReview` releases it if `StartAIReview` fails. `go test -race ./...` passes. QA must still see pending, then completed, without a reload.
- Owner/validation: developer; add a service test asserting `running` is true in the item response as soon as the collection is recorded for new item and price change cases; reviewer recheck; QA adds an item and observes pending then completed without reload.

### REV-002: Saving only the Claude token rewrites and resets the LeBoncoin session state

- Evidence: `src/components/SettingsModal.tsx:109` always sends `leboncoinSession: { value, revision }`; `internal/store/store.go` `saveLeboncoinSessionTx` resets `expires_at`, `revoked_at`, `last_attempt_*` and bumps revision for any non-null value.
- Impact and scenario: operator edits only the Claude token; the LeBoncoin expired/revoked warnings (FR-LBC-SET) disappear and other tabs get spurious `SESSION_CHANGED`.
- Criticality: critical, because it violates FR-CLT-SET-001 (session entry otherwise behaves per FR-LBC-SET-001..019) and the "preserve existing contracts" rule, and it destroys stored session status data on a normal, frequent action.
- Disposition: fix.
- Reason: the API already makes `leboncoinSession` optional; no contract change is needed.
- Required change: in `SettingsModal.save()`, omit `leboncoinSession` when its value equals the loaded value (and likewise omit `claudeToken` when unchanged, if not already so). Optionally also make the store a no-op for an unchanged value, but the client fix is mandatory.
- Specification decision: not applicable.
- Resolution: fixed and independently verified (review.md round 2). `SettingsModal.save` omits `leboncoinSession` when only the Claude token changed. When neither value changed, it still sends the session; this keeps the earlier single-entry behavior and is acceptable. `tests/claude-token-settings.spec.ts` is registered but has not been run yet; QA must run it.
- Owner/validation: developer; extend `tests/leboncoin-session-settings.spec.ts` (or a new spec) asserting the PUT body omits `leboncoinSession` when only the Claude token changes and the session warning remains; reviewer recheck.

### REV-003: `POST /ai-review` can return `CLAUDE_TOKEN_MISSING` while a token is saved

- Evidence: `internal/service/review.go:27,66-69` - `startReview` returns `false` for both "no token" and "already running"; the caller re-checks `reviewRunning`, and if the run finished meanwhile it returns `CLAUDE_TOKEN_MISSING`.
- Impact and scenario: rare; operator clicks refresh as a review finishes and sees "Configure a Claude token in Settings".
- Criticality: non-critical, because it requires a narrow race, causes no data loss, and a retry succeeds; it is still a contract violation of the API error table.
- Disposition: fix.
- Reason: trivial, low-risk correction of a contract mismatch; cheaper to fix than to track.
- Required change: `startReview` returns a distinct outcome (started / alreadyRunning / noToken) determined under `s.mu`; `RequestAIReview` maps it directly without the second `reviewRunning` check.
- Specification decision: not applicable.
- Resolution: fixed and independently verified (review.md round 2). `startReview` returns `reviewStarted`, `reviewAlreadyRunning` or `reviewNoToken`, decided under `s.mu`, and `RequestAIReview` maps each one directly.
- Owner/validation: developer; unit test for each outcome; reviewer recheck.

### REV-004: Automatic review after a failed first collection records a failed review

- Evidence: `service.go:282-283` starts the review for `newItem` regardless of collection outcome; `review.go` stores "The listing could not be retrieved from LeBoncoin." when details are nil.
- Impact and scenario: first collection blocked; a failed review is shown; the operator can retry with the refresh button.
- Criticality: non-critical, because behavior is explicitly covered by the specification and no data is lost.
- Disposition: reject.
- Reason: FR-LBC-AIR-001 triggers on item acceptance, and the edge-case list maps "listing unretrievable at review time" to FR-LBC-AIR-007 (understandable error with date, retry button available). The implementation matches. Deferring the review to "first successful collection" would be a new functional rule not in the specifications; the adjudicator cannot invent it. Remaining risk: an item whose first collection failed gets no automatic review until a price change or manual refresh, which the spec accepts. If the user wants different behavior, it is a new functional request.
- Specification decision: none required for this release (behavior specified).
- Resolution: rejected.
- Follow-up: none.

### REV-005: Images fetched by Anthropic by URL; one unfetchable image fails the whole review

- Evidence: `internal/claude/review.go` uses `source.type: "url"`; `client.go` maps other 4xx to `ErrBadResponse` ("Claude returned an unusable review"). Not verified against live API.
- Impact and scenario: if the CDN blocks Anthropic or a photo is removed, reviews fail with a misleading message.
- Criticality: pending (classified non-critical provisionally), because there is no evidence the failure occurs; if QA shows systematic failures it becomes critical (FR-LBC-AIR-002/007).
- Disposition: defer to QA evidence.
- Reason: changing to server-side download/base64 adds bandwidth, size limits and complexity without evidence of need. QA against the real API is the cheapest way to settle it.
- Required change if QA reproduces: download images server-side and send base64 (respecting size limits), or retry without the failing images and map the error to an accurate message.
- Specification decision: not applicable.
- Resolution: deferred pending QA.
- Follow-up: QA tester must run a real review on a listing with photos and record the outcome; adjudicator reclassifies on result.

### REV-006: "Older price" warning shown when the reviewed price is unknown

- Evidence: `src/components/AiReview.tsx:175` compares `latest.priceCents !== currentPriceCents` with `latest.priceCents` possibly null.
- Impact and scenario: listing with undetected price at review time shows a false "older price (unknown)" warning.
- Criticality: non-critical, because it is a misleading label only, on an uncommon path.
- Disposition: fix.
- Reason: FR-LBC-AIR-009 requires the indicator when the review was made at an older price; an unknown price is not evidence of a change. One-line fix.
- Required change: show the warning only when `latest.priceCents !== null && currentPriceCents !== null && latest.priceCents !== currentPriceCents`.
- Specification decision: not applicable.
- Resolution: fixed and independently verified (review.md round 2). The warning now requires `latest.priceCents !== null` (`AiReview.tsx:175`).
- Owner/validation: developer; reviewer recheck.

### REV-007: Listing text passed to Claude without prompt-injection framing

- Evidence: `internal/claude/review.go` `reviewPrompt` embeds seller text with no untrusted-data instruction.
- Impact and scenario: a seller can attempt to steer review content; output stays schema-validated and only advisory.
- Criticality: non-critical, because impact is limited to the advisory text and enums of one review, with no data or credential exposure.
- Disposition: fix.
- Reason: a technical hardening that changes no specified behavior; a single sentence in the system prompt (and delimiting listing fields) is low risk and improves fidelity of FR-LBC-AIR-003 output. Not a new functional requirement.
- Required change: add a system instruction that listing fields are untrusted seller-provided data, not instructions, and wrap them in clear delimiters.
- Specification decision: not applicable.
- Resolution: fixed and independently verified (review.md round 2). A system instruction declares `<listing_data>` untrusted. The data is `json.MarshalIndent` output, whose HTML escaping stops a seller from closing the tag.
- Owner/validation: developer; existing tests updated if they assert prompt content; reviewer recheck.

### REV-008: Minor formatting regressions

- Evidence: `src/App.tsx` `const [refreshingItemIds,setRefreshingItemIds]`, `src/api/items.ts` `ItemStatus,PriceObservation`.
- Impact and scenario: noise in diff only.
- Criticality: non-critical, because there is no behavior impact.
- Disposition: fix.
- Reason: unrelated formatting changes violate the narrow-change-scope rule; restoring them reduces diff noise.
- Required change: restore original spacing.
- Specification decision: not applicable.
- Resolution: fixed and independently verified (review.md round 2). The spacing is restored in `src/App.tsx:53` and `src/api/items.ts:2`.
- Owner/validation: developer; reviewer confirms diff no longer touches these lines beyond needed changes.

### REV-009: SQLite read performed while holding the service-wide mutex

- Evidence: `internal/service/review.go:92-107`. `reserveReview` calls `s.store.ClaudeToken` while holding `s.mu`. The same mutex guards `inFlight` (every collection start and end) and `reviewing` (every details GET through `reviewRunning`). With `busy_timeout=5000`, a locked database can keep `s.mu` held for up to 5 s. The REV-003 fix introduced this.
- Impact and scenario: while SQLite is write-locked, detail polls and collection bookkeeping can wait up to 5 s. Only latency is affected; results stay correct.
- Criticality: non-critical. It causes no incorrect data, no contract violation and no security exposure. It needs a locked database to happen, and the 5 s busy timeout limits the wait.
- Disposition: fix.
- Reason: the change is small and local. It follows the go-backend-architecture guidance to keep critical sections short, and it removes a regression introduced within this change. Reading the token before taking the lock is safe. If a token save races with the read, the review uses the token it read at start. That matches the specified edge case: a review "finishes or fails with the token it started with". Rejection handling already checks the token revision.
- Required change: in `reserveReview`, read `ClaudeToken` before `s.mu.Lock()`, and return the error or `reviewNoToken` when no token is saved. Keep only the `reviewing[id]` check-and-set under `s.mu`. Keep the REV-003 outcome mapping unchanged.
- Specification decision: not applicable.
- Resolution: fixed (reviewer round 3).
- Owner/validation: developer. `go test -race ./...` passes, including the REV-003 outcome tests; reviewer recheck.
- Resolution evidence (reviewer recheck, round 3): fixed and verified. `internal/service/review.go:93-108` reads `ClaudeToken` before `s.mu.Lock()` and returns the error or `reviewNoToken` before locking. Only the `reviewing[id]` check-and-set runs under `s.mu`. The outcome mapping is unchanged. `go test -race ./internal/...` passes.

## Release readiness

No critical finding remains open: REV-001 and REV-002 are fixed and independently verified. REV-003, REV-006, REV-007 and REV-008 are also fixed and verified. REV-004 is rejected because the behavior is specified. REV-005 is deferred until QA tests against the live API; it becomes blocking if QA reproduces systematic failures. REV-009 is non-critical but must be fixed, and the reviewer must recheck it, before commit. No functional questions are open for the user. QA still needs to run the new Playwright specs and test the automatic review flow.

# Review: LeBoncoin old price (issue #42)

Reviewer: independent reviewer agent (did not write the code).
Revision reviewed: base commit `513035c` + uncommitted working tree (modified and untracked files listed by `git status` on 2026-10-05).
Inputs: `functional.md`, `technical.md` (FR/TS-LBC-OLD-PRICE-*), `API_SPECIFICATION.md` "LeBoncoin old price", full diff and new tests.

Verification run by reviewer: `go test ./...` passes (claude, httpapi, leboncoin, service, store); `npm run build` (typecheck + Vite) succeeds.

## Summary

Implementation matches the specifications and API contract: old price parsed defensively only on success branches, cleared in `collectReserved` unless `newItem && success`, inserted in the same transaction with `is_old_price = 1` and `''` for undated, idempotent migration via generalised `ensureColumn` (recorded refactor), `timestamp: null` / `oldPrice` in JSON, UI shows plain-text "Old price", list keys use an index fallback. All remaining `Timestamp` uses were checked (`statusFor`, AI prompt, store scans); Amazon second-hand detections keep non-null `time.Time` and a separate TS type. No correctness defects found. Only low-severity findings below.

## Findings

### REV-001 — AI prompt labels undated old prices "old price" (unspecified behavior)
- Requirement: unspecified (functional spec only says the old price triggers no additional review).
- Location: `internal/claude/review.go:96-100`.
- Evidence: `PriceHistory` now returns undated rows with `Timestamp == nil`; the change writes `date: "old price"` instead of dereferencing nil.
- Assessment: the nil check is required (otherwise a nil-pointer panic on reviews of new items with an old price). Sending the old price to Claude in the history is a direct consequence of FR-LBC-OLD-PRICE-005 ("part of the item's price history"). However, the label text and the fact that dated old prices are not distinguished (the `PriceHistory` query in `internal/store/ai_reviews.go` does not select `is_old_price`) are not described in the technical spec.
- Expected: behavior documented. Actual: undocumented.
- Suggested action: accept and record in `technical.md` (Backend) that undated old prices are sent to the AI review prompt with date `"old price"`; no code change needed.
- Provisional severity: low.

### REV-002 — Unreachable null-timestamp branch for `latestPrice`
- Requirement: TS-LBC-OLD-PRICE-005 / technical spec states `latestPrice` is never an undated old price.
- Location: `src/components/ItemDetail.tsx:147-149`; `internal/store/store.go:672` (`price.Timestamp == nil` → `stale`).
- Evidence: the old price is only inserted together with a newer collected observation, and dated old prices not before the collection time are made undated (`service.go:314`), so `latestPrice` always has a timestamp.
- Impact: none at runtime; small defensive code for an impossible state. Not a defect.
- Suggested action: keep (type-safe handling of the nullable type) or simplify; preference only.
- Provisional severity: informational.

### REV-003 — Feature index status still "specified"
- Location: `doc/FUNCTIONAL_SPECIFICATIONS.md` last table row ("LeBoncoin old price ... specified").
- Expected: "implemented" once the change lands, as for other delivered features in the same table.
- Suggested action: update the status in the delivery commit.
- Provisional severity: low.

### REV-004 — Real `old_price` shape unverified (open risk, not a defect)
- Requirement: technical spec "Source shape of the old price".
- Evidence: parser accepts a number or one-element array in euros; no real-listing fixture exists. If LeBoncoin uses cents, an object, or a string, nothing is recorded (safe failure, FR-006), so the feature would silently do nothing.
- Suggested action: QA must inspect a real listing with a crossed-out price and record the shape, as the technical spec requires.
- Provisional severity: low (tracked for QA).

## Checked without findings
- FR-002/007: `!newItem || result != "success"` clears fields; service tests cover refresh and failed-first-collection.
- FR-005 ordering: `''` sorts first in `ORDER BY observed_at`; LAG/period CTE and `LIMIT 3` include old price correctly; tests cover undated and dated ordering.
- FR-008: cascade delete unchanged and tested.
- FR-009: Amazon collector never sets the fields.
- Migration: `ALTER TABLE ... ADD COLUMN` with constant default and CHECK is valid in SQLite; idempotence tested.
- Frontend keys: `${timestamp}-${index}` avoids duplicate/`null` keys.
- Accessibility: label is plain text, no `<time>` element.

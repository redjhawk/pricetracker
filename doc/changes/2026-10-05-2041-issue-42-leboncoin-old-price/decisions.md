# Review decisions: LeBoncoin old price (issue #42)

Reviewed specification revisions: `doc/specifications/leboncoin-old-price/functional.md`, `technical.md`, `API_SPECIFICATION.md` "LeBoncoin old price" (working tree, 2026-10-05).
Reviewed code revision: base `513035c` + uncommitted working tree.
Reviewer: independent reviewer agent ([review.md](review.md)).
Adjudicator: independent review adjudicator agent (distinct from developer and reviewer).

## Findings and decisions

### REV-001: AI prompt labels undated old prices "old price" (unspecified behavior)

- Evidence: `internal/claude/review.go:96-100` writes `date: "old price"` when `Timestamp == nil`; `PriceHistory` (`internal/store/ai_reviews.go`) does not select `is_old_price`. The technical spec did not describe this.
- Impact and scenario: a new LeBoncoin item with an undated old price is reviewed by Claude; the nil check prevents a nil-pointer panic. Including the old price follows from FR-LBC-OLD-PRICE-005 (part of the price history). The only effect is prompt wording.
- Criticality: non-critical, because the behavior is correct, safe and consistent with the functional requirement; only documentation was missing. No user-visible or data impact.
- Disposition: fix (documentation only).
- Reason: unspecified behavior must be recorded. No code change is warranted: the label is meaningful and no requirement asks to distinguish dated old prices for the AI.
- Specification decision: not applicable (technical detail).
- Resolution: fixed by the adjudicator — "AI review prompt" bullet added to the Backend section of `doc/specifications/leboncoin-old-price/technical.md`.
- Follow-up: none.

### REV-002: Unreachable null-timestamp branch for `latestPrice`

- Evidence: `src/components/ItemDetail.tsx:147-149`, `internal/store/store.go:672`. The old price is only inserted with a newer collected observation, and dated old prices not before the collection time are stored undated (`service.go:314`), so `latestPrice` always has a timestamp.
- Impact and scenario: none at runtime.
- Criticality: non-critical, because the branch is unreachable and harmless.
- Disposition: reject.
- Reason: `timestamp` is nullable in the API contract and TS type; handling null is needed for type safety (otherwise a non-null assertion would be required) and in Go prevents a panic should the invariant change. Removing it trades robustness for a few lines. Remaining risk: none.
- Specification decision: not applicable.
- Resolution: rejected; code kept as is.
- Follow-up: none.

### REV-003: Feature index status still "specified"

- Evidence: `doc/FUNCTIONAL_SPECIFICATIONS.md` feature table row "LeBoncoin old price ... specified".
- Impact and scenario: documentation would misstate delivery status after merge.
- Criticality: non-critical, because documentation-only.
- Disposition: fix.
- Reason: the feature is implemented and lands in the same PR; other delivered features use "implemented".
- Specification decision: not applicable.
- Resolution: fixed by the adjudicator; status set to "implemented".
- Follow-up: none.

### REV-004: Real `old_price` shape unverified (open risk)

- Evidence: the parser accepts a number or a one-element array in euros; no real-listing fixture. The technical spec ("Source shape of the old price") already records this limitation and assigns confirmation to QA.
- Impact and scenario: if LeBoncoin uses another shape (object, string), nothing is recorded (safe failure, FR-LBC-OLD-PRICE-006) and the feature is silently ineffective. If the value were numeric cents, recorded old prices would be 100x too large — the main residual risk.
- Criticality: non-critical for review completion, because no crash, data loss or regression of existing behavior results, and verification needs live data only QA can obtain. Becomes a fix item if QA observes a different shape or unit.
- Disposition: defer to QA (stage 7).
- Reason: cannot be resolved by code inspection; release acceptable since the limitation is documented and failure is mostly safe.
- Specification decision: not applicable.
- Resolution: deferred.
- Follow-up: QA tester must inspect a real LeBoncoin listing with a crossed-out price and record the `old_price` shape, unit and any date field; any mismatch is a new finding routed to the developer.

## Release readiness

No critical findings and no open functional questions. REV-001 and REV-003 fixed (documentation), REV-002 rejected with rationale, REV-004 deferred to QA with an explicit verification condition. Reviewer verification: `go test ./...` and `npm run build` pass. Ready for stage 8; QA outcome pending.

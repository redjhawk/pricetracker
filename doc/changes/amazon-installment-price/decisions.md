# Review decisions: Amazon instalment price (issue #45)

Reviewed specification revisions: [functional.md](../../specifications/amazon-price-detection/functional.md) (ready), [technical.md](../../specifications/amazon-price-detection/technical.md) (ready), working tree on branch `ai-dev/issue-45-20261005-2049`.
Reviewed code revision: base `513035cdd41f7267aa08f563c711ad7d95030e71` plus uncommitted diff to `internal/amazon/collector.go` and new `internal/amazon/collector_test.go`.
Reviewer: independent reviewer agent ([review.md](review.md)).
Adjudicator: independent review adjudicator agent (separate from developer and reviewer).

The review reported no findings. It listed three non-blocking observations, which are adjudicated below as OBS items. The adjudicator checked the diff against the specifications: the regex matches the technical text exactly, both new regexes are compiled at package level, and `go test ./internal/amazon/` passes.

## Findings and decisions

### OBS-001: Instalment wordings outside the `x<N> <month>` form are not detected

- Evidence: `instalmentText` in `internal/amazon/collector.go` only matches text starting with `x<digits> <month word>`. Wordings such as "4 x 128,65 €", "x4 mensualités" or "4 mensualités de" are not matched by the text rule. The container rule (`installment|inemi` id/class) still catches them when the amount is rendered inside Amazon's instalment widget.
- Impact and scenario: if an Amazon page shows an instalment amount with a different wording, outside an `installment`/`inemi` container and outside an `a-price` wrapper that the existing `installment` class filter excludes, that amount could still be a price candidate. It would only be stored if it also won the existing candidate ordering.
- Criticality: non-critical. The reported case (issue #45, "x4 mois") and the specified marketplace month words are handled and tested. The remaining gap needs three conditions at once: a different wording, no instalment container, and winning the candidate ordering. There is no evidence of such a page. The technical specification deliberately limits detection to this pattern, which is a reasonable implementation of FR-AMAZON-PRICE-002's example. This is a coverage limit, not a defect against the specification.
- Disposition: defer.
- Reason: widening the regex without real page samples risks false positives that drop real prices, which would break FR-AMAZON-PRICE-001/004. The fix is cheap to add later, once a real page with another wording shows up. Remaining risk: low. A wrong price would be visible in price history and could be reported like issue #45.
- Specification decision: not applicable. No functional ambiguity: FR-002 requires that instalment amounts are not stored, and the technical stage chose the detection rule.
- Resolution: deferred.
- Follow-up: technical specifier/developer, if a real listing with another wording is reported. Add the wording to `instalmentText` together with a fixture.

### OBS-002: Instalment containers opened more than 2,000 characters before the price are not detected

- Evidence: `isInstalmentPrice` only inspects `openTags(prefix)`, and the prefix is the existing 2,000-character window. That window is exactly what the technical specification (Backend rule 2) defines.
- Impact and scenario: a very large instalment widget whose opening tag is more than 2,000 characters before an inner `a-price` would only be caught if the text rule matches. Amazon instalment amounts are normally followed by "x4 mois"-style text, so the text rule covers them independently.
- Criticality: non-critical. The two rules overlap, the behavior matches the specification, and there is no evidence of such markup. Even if missed, the amount must still win the candidate ordering.
- Disposition: defer.
- Reason: a larger window costs more parsing per candidate and makes it more likely that an unrelated open ancestor (e.g. a page-level container whose class contains "inemi") wrongly drops the real price. Keeping the specified window is the safer tradeoff. Remaining risk: low.
- Specification decision: not applicable.
- Resolution: deferred.
- Follow-up: technical specifier, only if a real page shows the gap.

### OBS-003: Pre-existing `regexp.MustCompile` calls inside the `productPrice` loop

- Evidence: the wrapper-filter regex (`a-text-price|price-per-unit|...`) in `productPrice` and the `EUR` regex in `parseEuroPrice` are compiled on every call. This code predates the change. The new code only touched the line next to it, and the new regexes are compiled at package level as specified.
- Impact and scenario: minor CPU overhead per price candidate during collection. No correctness impact.
- Criticality: non-critical. This is a performance/style suggestion, not a defect, and it was not introduced by this change.
- Disposition: reject for this change (out of scope).
- Reason: the technical specification states "No refactoring", and AGENTS.md requires a narrow change scope with any refactoring recorded separately. Hoisting these regexes is an unrelated refactor that belongs in its own PR. The collection workload is small (a few regex compilations per page), so leaving the code as it is has negligible cost.
- Specification decision: not applicable.
- Resolution: rejected for this change. Unchanged code.
- Follow-up: optional separate refactoring PR to hoist the regexes to package level (owner: technical specifier/developer). Not tracked as a blocker.

## Release readiness

- Open blocking (critical) findings: none.
- Unresolved functional questions: none.
- Deferred: OBS-001 and OBS-002 (non-critical coverage limits that match the specification). Rejected for this scope: OBS-003 (pre-existing, unrelated refactor).
- Checks: `go test ./internal/amazon/` passes; reviewer reports `go vet` is clean.
- QA: pending (stage 7, after the PR is opened). The change may proceed to the commit/PR stage.

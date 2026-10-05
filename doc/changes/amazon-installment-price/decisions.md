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

## QA findings (stage 7)

### QA-F-001: Live issue #45 page stores the instalment amount (128,65 €) instead of the price (514,63 €)

- Evidence: `qa.md` QA-001/QA-002. On https://www.amazon.fr/dp/B0F99B78JN, the item was stored at 128,65 €. The live markup shows "Ou 128,65€ x4 (0,0% de frais inclus)": an `a-price` inside `span#price-block-message.price-block-message > span#price-block-amount-prefix` ("Ou ") `+ span#price-block-amount`. There is no "mois" text and no container whose name mentions instalments, so neither the container rule nor the text rule matches. The unit fixtures do not reproduce this markup, so the tests pass while the live page fails.
- Impact and scenario: the exact page reported in issue #45 still stores and shows the instalment amount. This defeats the purpose of the change.
- Criticality: **critical**. It violates FR-AMAZON-PRICE-001/002 on the issue's own reference page.
- Disposition: **fix** (blocks release).
- Required changes:
  1. Technical specification (`doc/specifications/amazon-price-detection/technical.md`): exclude price candidates inside the `price-block-message` / `price-block-amount` elements (match by id or class: `price-block-message`, `price-block-amount`, `price-block-amount-prefix`).
  2. Technical specification: treat an `x<N>` multiplier with no month word as an instalment marker. That is, after tags are stripped and whitespace is trimmed, the text following the price starts with `x\s*\d+` followed by a non-digit or the end of the text (e.g. "x4 (0,0% de frais inclus)").
  3. Developer: implement both rules and add a test fixture copied from the live markup (the price-block-message structure with "Ou 128,65€ x4 (0,0% de frais inclus)" next to `.priceToPay` 514,63 €). The test must expect 514,63 €.
  4. QA: retest QA-001 against the live page after the fix.
- Reason: Amazon's live instalment widget uses a structure and wording that the specified rules did not anticipate. Fixing this in the specification first keeps the code limited to specified behavior. The two rules are independent, so either one alone catches this page.
- Specification decision: the technical specification must be updated before development. No functional change is needed, because FR-001/002 already require this outcome.
- Resolution: fix implemented, pending live QA retest of QA-001. The technical specification (TS-AMAZON-PRICE-001) was updated. `instalmentText` is now `^\s*(€)?\s*x\s*\d+(\D|$)`, and `instalmentContainer` adds `price-block-message|price-block-amount` (this also covers `price-block-amount-prefix`). Two fixtures copied from the live markup (widget after and before `.priceToPay`) expect 514,63 €, and `go test ./internal/amazon/` passes. The independent re-review found no blocking issue (see REV-QA-001). This finding closes only when QA-001 passes on the live page.

## QA fix re-review decisions

### REV-QA-001: The bare `x<N>` rule can skip a real price when the next visible text starts with `x<digit>`

- Evidence: [review.md](review.md), "QA fix re-review". `isInstalmentPrice` matches `^\s*(€)?\s*x\s*\d+(\D|$)` case-insensitively against the first 300 stripped characters after the `a-price` element. That text can belong to a following sibling or block. If it starts with something like "X100V" or "x3 pack", the real price candidate is skipped.
- Impact and scenario: the main price is skipped, and detection falls back to another candidate or to `price_not_found`. A wrong price is stored only if the fallback candidate is itself wrong. A missing price is visible to the user through the existing `price_not_found` path, so this is not silent data corruption.
- Criticality: non-critical. To trigger it, the visible text right after the main price element must start exactly with `x` followed by a digit. In Amazon's price block, the elements after `.priceToPay` hold price-related content (savings, unit price, delivery, the instalment widget), not product or model names. Pack and model names usually appear in the title, before the price block. There is no evidence of real markup that triggers this. The broader rule was a deliberate QA-F-001 decision, because the live widget has no month word.
- Disposition: reject (accept the documented risk) for this change.
- Reason: the reviewer's alternatives are worse or unproven:
  - Requiring a month word brings back the QA-F-001 failure ("x4 (0,0% de frais inclus)").
  - Requiring "(" or "%" after the number fits our only sample too closely, so it would break on other marketplaces or wordings (for example "x4 sans frais").
  - Limiting the text to a block boundary adds parsing complexity, and there is no observed failing case to test it against.

  The container rule catches the live widget on its own, so the text rule acts as a second safeguard. Remaining risk: low, and visible when it happens.
- Specification decision: not applicable. TS-AMAZON-PRICE-001 already specifies this rule.
- Resolution: rejected for this change. No code change.
- Follow-up: technical specifier/developer, only if someone reports a real listing where a valid price is skipped. Then narrow the text rule and add a fixture from that page.

## Release readiness (current)

- Open blocking (critical) findings: QA-F-001 remains open until it is retested. The fix is implemented in the specification and code, and the re-review passed. It closes when the live QA retest of QA-001 passes.
- Unresolved functional questions: none.
- Deferred: OBS-001, OBS-002. Rejected for this scope: OBS-003, REV-QA-001 (risk accepted and documented).
- Checks: `go test ./internal/amazon/` passes.
- Gate: the QA fix may be committed and pushed to the existing PR branch. Completion requires QA-001 to pass on https://www.amazon.fr/dp/B0F99B78JN, with the stored price at 514,63 €.

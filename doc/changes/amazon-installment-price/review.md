# Review: Amazon instalment price (issue #45)

Reviewer: independent reviewer agent (not the implementer).
Reviewed revision: base commit `513035cdd41f7267aa08f563c711ad7d95030e71` plus the uncommitted diff to `internal/amazon/collector.go` and the new file `internal/amazon/collector_test.go`.
Specifications: [functional](../../specifications/amazon-price-detection/functional.md), [technical](../../specifications/amazon-price-detection/technical.md).
Verification evidence: `go test ./internal/amazon/` passes; `go vet ./internal/amazon/` reports nothing.

## Result

No findings.

## What was checked

- FR-AMAZON-PRICE-001/002: the issue #45 fixture (514,63 € then "Ou 128,65 € x4 mois") returns 51463. The instalment `a-price` is dropped because the visible text right after the element starts with "x4 mois". Instalment before the price, an `inemi` container, and German "x4 Monate" are also covered.
- FR-AMAZON-PRICE-003: an instalment-only fixture returns not found, so the existing `price_not_found` path applies.
- FR-AMAZON-PRICE-004: when no instalment marker is present, the filters and candidate ordering are unchanged. A normal-price regression fixture passes.
- Technical spec conformance: the text rule uses exactly the specified regex on the first 300 characters of text after the element. The container rule checks `id`/`class` for `installment|inemi` on the unclosed tags in the existing 2,000-character prefix. Both regexes are compiled at package level. The change is limited to `productPrice` plus three small helpers. There is no API, persistence or frontend change.
- False positives (could a real price be dropped?):
  - The text rule is anchored at the start of the text after the price element. Text after a main price ("Livraison…", "Ou 128,65 €…", another price) does not begin with `x<digits> <month word>`, so it does not match.
  - The container rule only matches tags that are still open around the price. A closed instalment widget before the main price does not affect it (covered by the "instalment container" test).
  - The `a-price` wrapper itself is included in the open-tag stack, but the existing wrapper filter already excludes `installment` classes, so this changes nothing.
- Robustness: if the prefix window starts mid-tag, the leftover text is parsed as text and causes no problem. `elementEnd` stays inside a 20,000-character window and falls back to the window end. Unmatched closing tags are ignored.

## Non-blocking observations (not findings)

- Spec-limited coverage: wording that does not use the `x<N> <month>` form (e.g. "4 x 128,65 €", "x4 mensualités") and instalment containers opened more than 2,000 characters before the price are not detected. This matches the technical specification, so no action is needed for this change.
- Pre-existing, not caused by this change: `regexp.MustCompile` calls inside the `productPrice` loop recompile on every candidate.

## QA fix re-review (QA-F-001, TS-AMAZON-PRICE-001)

Scope: uncommitted diff to `internal/amazon/collector.go`, `collector_test.go` and `technical.md`. The implementation matches TS-AMAZON-PRICE-001: the text regex is replaced by `^\s*(€)?\s*x\s*\d+(\D|$)`, the container regex adds `price-block-message|price-block-amount`, both are compiled at package level, and both live-markup orders are tested. `go test ./internal/amazon/` passes.

### REV-QA-001: The bare `x<N>` rule can skip a real price when the next visible text starts with a model or pack name

- Evidence: `isInstalmentPrice` matches against the first 300 characters of the stripped text that follows the `a-price` element, case-insensitively. That text can come from the next sibling or block, not only the instalment suffix. If the text after `.priceToPay` starts with something like "X2 ...", "x3 pack" or "X100V", the rule matches and the real price is skipped. Pack text such as "x2" usually appears in the title, before the price block, so it does not trigger the rule. The real risk is limited to a following element whose text starts with `x<digit>`.
- Impact: if this happens, the price falls back to another candidate or to `price_not_found`. It is not a silent wrong price unless a later candidate is itself wrong.
- Criticality (reviewer's view): non-critical, PLAUSIBLE. There is no evidence of real Amazon markup that does this. The adjudicator decided on this pattern in QA-F-001.
- Suggestion (optional): limit the text rule to the text before the next block-level boundary, or require a month word or "(" / "%" after the number. Otherwise, accept the risk as documented.

### Container rule: no finding

The container rule only checks opening tags that are still unclosed in the 2,000-character prefix. The sibling `price-block-message` widget closes before `.priceToPay` (the "before price" test covers this), so it cannot exclude the main price. The main price would only be excluded if Amazon wrapped `.priceToPay` itself in an element whose id or class contains `price-block-message` or `price-block-amount`. Neither the QA evidence nor known markup shows this. Residual risk: low, and QA's live retest (QA-001) will confirm it.

Result: one non-critical finding (REV-QA-001). No blocking findings.

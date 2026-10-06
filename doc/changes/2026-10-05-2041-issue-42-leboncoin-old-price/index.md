# LeBoncoin old price

Stage: review decisions complete (next: commit/PR, then QA)
State: functional, technical and API specifications ready.

User request (GitHub issue #42, "Add older price for leboncoin when available"): record the listing's `old_price` as a historical price only when a LeBoncoin item is added; use its date when available, otherwise show "Old price"; no new column; update the functional specifications file.

## Subjects

1. LeBoncoin old price — [functional](../../specifications/leboncoin-old-price/functional.md) FR-LBC-OLD-PRICE-001–009; ready. Technical specification: ready.

## Stage results

1. Functional specification — ready; `doc/FUNCTIONAL_SPECIFICATIONS.md` updated (sections 6, 8 FR-31, 9, 15).
2. Technical specification — ready ([technical.md](../../specifications/leboncoin-old-price/technical.md), TS-LBC-OLD-PRICE-001–005). Design: `price_observations.is_old_price` flag; undated old price stored with empty `observed_at` (sorts oldest); old price kept only for the add-time successful collection. Rationale: reuses the single price history (no new column, per user), no query rewrites. Refactor: `ensureItemColumn` generalised to any table (needed for the migration). Limitation: `old_price` shape and any date field unverified (no fixture); parsed tolerantly, QA to confirm on a real listing.
3. API specification — `API_SPECIFICATION.md` section "LeBoncoin old price": observations gain `oldPrice` boolean; `timestamp` nullable for undated old prices. Other contracts preserved.
4. Development — implemented (working tree).
5. Review — [review.md](review.md): no correctness defects; 4 low/informational findings.
6. Review decisions — [decisions.md](decisions.md): no critical findings; REV-001 and REV-003 fixed (documentation), REV-002 rejected, REV-004 deferred to QA. No blocker.
7–8. Pending.

## User decisions

From issue #42: read only at add time; use the date when present, else show "Old price"; stored as a historical price, not a new column.

## Unresolved questions

None.

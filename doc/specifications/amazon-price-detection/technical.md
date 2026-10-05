# Technical specification: Amazon price detection

Status: ready
Functional specification: [functional.md](functional.md)

## Requirement mapping

| Functional ID | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- |
| FR-AMAZON-PRICE-001, 002 | Skip instalment candidates in `productPrice` (see Backend). | `internal/amazon/collector.go` | Fixture with 514,63 € plus "128,65 € x4 mois" returns 51463. |
| FR-AMAZON-PRICE-003 | No candidate remains, so `productPrice` returns `false` and the existing `price_not_found` path applies. | same | Instalment-only fixture returns not found. |
| FR-AMAZON-PRICE-004 | Existing filters and ordering unchanged. | same | Existing tests plus a normal-price regression fixture pass. |

## Frontend

No change: the stored price is displayed as today.

## Backend

In `productPrice` (`internal/amazon/collector.go`), after a candidate passes the existing wrapper filters, ignore it when either:

1. The visible text content following its `a-price` element (tags stripped, first ~300 characters) matches, case-insensitive:
   `^\s*(€)?\s*x\s*\d+\s*(mois|monat|monate|mes|meses|mesi|maand|maanden|months?)`
2. The near prefix (the existing 2,000-character prefix window) contains an opening element whose `id` or `class` contains `installment` or `inemi` that encloses the price.

Compile the new regexes once at package level. No persistence, migration or API changes.

## API

No contract change: `API_SPECIFICATION.md` is unaffected; only the parsed value changes.

## Scope and refactoring

Change limited to `productPrice` and its tests. No refactoring.

## Verification and unresolved questions

- Unit tests in `internal/amazon/collector_test.go` with HTML fixtures: issue #45 reproduction (514,63 € + 128,65 € x4 mois → 51463), instalment-only (→ not found), normal price regression, one non-French marketplace wording (e.g. "x4 Monate").
- `go test ./...`.
- Unresolved questions: none. Live page fetch is unavailable from this environment; fixtures model the reported text.

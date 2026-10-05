# Functional specification: Amazon price detection

Status: ready
Owner: functional specification agent
User decision/reference: issue #45 (Amazon.fr https://www.amazon.fr/dp/B0F99B78JN shows 514,63 € with "Ou 128,65 € x4 mois (2,3% de frais inclus)"; the app stored 128,65 €; the user wants the real product price).

## Purpose and scope

When an Amazon listing is collected, the stored price is the product's purchase price. Applies to supported marketplaces (fr, de, es, it, nl, be). Excluded: any other change to Amazon collection.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMAZON-PRICE-001 | Listing collected | The stored price is the product's purchase price. | Given the issue #45 page, when collected, then 514,63 € is stored. |
| FR-AMAZON-PRICE-002 | Page shows an instalment / monthly-payment offer (e.g. "128,65 € x4 mois", per-month amounts in any supported marketplace language) | The instalment amount is never stored as the price. | Given a page with price 514,63 € and instalment 128,65 € x4, then 128,65 € is not stored. |
| FR-AMAZON-PRICE-003 | Only instalment amounts are present | Collection result is `price_not_found`. | Given a page whose only price is an instalment amount, then the result is `price_not_found`. |
| FR-AMAZON-PRICE-004 | Page without instalment offer | Existing behavior is unchanged. | Given a normal listing, the same price as before is stored. |

## States and corner cases

Instalment offer before or after the main price; instalment-only page (FR-003); pages without any instalment widget (FR-004).

## Open questions

None.

## Traceability

Technical specification: [technical.md](technical.md). Change record: [../../changes/amazon-installment-price/index.md](../../changes/amazon-installment-price/index.md).

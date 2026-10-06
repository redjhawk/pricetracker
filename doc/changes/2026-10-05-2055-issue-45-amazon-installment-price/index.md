# Change: Amazon instalment price (issue #45)

Stage: review decisions done (no critical findings); next: commit and PR, then QA.

## Scope

Backend-only fix: Amazon collection must not store instalment / monthly-payment amounts as the product price.

## Links

- Functional: [../../specifications/amazon-price-detection/functional.md](../../specifications/amazon-price-detection/functional.md)
- Technical: [../../specifications/amazon-price-detection/technical.md](../../specifications/amazon-price-detection/technical.md)
- Review: [review.md](review.md)
- Review decisions: [decisions.md](decisions.md)

## API contract

Unchanged: no endpoint, field or status changes; only the value parsed from Amazon pages is corrected.

## User decisions

None needed; the issue states the required behavior.

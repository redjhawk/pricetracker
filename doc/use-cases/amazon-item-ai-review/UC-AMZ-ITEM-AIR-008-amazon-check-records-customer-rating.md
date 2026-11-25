# UC-AMZ-ITEM-AIR-008 — An Amazon check records the customer rating

**Primary actor:** PriceFollower scheduler, or the operator choosing **Refresh price** or **Refresh all prices**
**Supporting systems:** Real web browser, Amazon, SQLite store
**Trigger:** An Amazon check (scheduled, **Refresh price**, **Refresh all prices** or the first check of a new item) gets a product page as a usable Amazon answer.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-030, FR-AMZ-ITEM-AIR-031, FR-AMZ-ITEM-AIR-021

## Preconditions

- An Amazon item is tracked and checked as in UC-AMZ-HCHK-001 or UC-AMZ-HCHK-003.

## Main flow

1. The check reads the product page and records its price and second-hand result as today (FR-AMZ-HCHK-002).
2. The page shows 4.5 out of 5 stars and 1,234 ratings; the check also records the customer rating 4.5 and the number of ratings 1234.
3. The operator later chooses **Refresh AI review** (UC-AMZ-ITEM-AIR-003); the review receives the rating 4.5 and 1234 ratings.

## Alternatives and errors

- **Page shows no rating:** the rating is recorded as absent; the price is still recorded and the next review states that the rating is unknown.
- **Rating cannot be read:** the check is not a failure because of it, and its price and second-hand result are unchanged.
- **Check fails (retrieval error):** the last recorded rating is kept.
- **Item added before this feature:** its rating is unknown until its next check that records it; a review made before that check is sent the rating as absent.

## Postconditions

- The item has the rating and number of ratings of its latest successful check; the customer rating is shown only through the AI review.

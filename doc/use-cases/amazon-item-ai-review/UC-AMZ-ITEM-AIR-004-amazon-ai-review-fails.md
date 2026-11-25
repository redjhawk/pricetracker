# UC-AMZ-ITEM-AIR-004 — An AI review of an Amazon item fails

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Claude, SQLite store
**Trigger:** A review requested with **Refresh AI review** fails: the token is rejected, the usage limit is reached, the network fails or Claude returns an error.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-014, FR-AMZ-ITEM-AIR-017, FR-AMZ-ITEM-AIR-025

## Preconditions

- The preconditions of UC-AMZ-ITEM-AIR-003 hold; the item may already have a successful review.
- Earlier Amazon checks may have failed, for example 4 consecutive failures.

## Main flow

1. The operator chooses **Refresh AI review** and the review is in progress (UC-AMZ-ITEM-AIR-003, steps 1–4).
2. The review fails.
3. The section shows and announces an understandable error with its date/time. The error never contains the token.
4. The latest successful review stays visible with its own date/time and reviewed price.
5. The **Refresh AI review** button is enabled again so that the operator can retry.

## Alternatives and errors

- **No previous review:** the section shows the error and that no review exists yet.
- **Token rejected:** the Claude token warning in Settings is updated.
- **Retry succeeds:** UC-AMZ-ITEM-AIR-003 continues from step 5.

## Postconditions

- No new review is stored; the history is unchanged.
- The failure is a review failure, not an Amazon check failure: the consecutive Amazon failure count is unchanged and no Amazon item is stopped. After 4 failed Amazon checks and a failed review, every Amazon item is still running.

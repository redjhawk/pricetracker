# UC-TI-01 — Review tracked items

**Primary actor:** Operator
**Supporting system:** PriceFollower API
**Trigger:** The operator opens the tracked-items page.

## Preconditions

- The application is available.

## Main flow

1. The page requests the tracked-item collection from `GET /api/v1/items`.
2. The system returns the shared collection, ordered by most recently added item first.
3. The page shows each item's title or fallback label, thumbnail when available, platform, listing ID, status, and row actions.
4. A single **Prices** column shows up to three recent consecutive price periods, newest first. The latest price is shown once as the newest period, with its most recent successful observation time.
5. Repeated successful checks at the same price update that period's displayed time. A changed price starts a new period, including a return to a price seen before an intervening change.
6. For Amazon items, the page shows the latest qualifying Amazon-sold second-hand offer or its pending, not-found, or check-error state.

## Alternatives and errors

- If the collection is empty, the page explains that no items are tracked and offers the add-item action.
- While the request is pending, the page shows a loading state.
- If loading fails, the page shows an error and a retry action.
- If an item has no successful price observation, its price cell says that the first price is awaited.
- A stale or failed check does not replace the last successful price; the item status communicates the collection state.

## Postconditions

- The operator can review the current server-provided state without changing tracked data.

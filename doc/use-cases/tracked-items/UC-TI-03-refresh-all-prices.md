# UC-TI-03 — Refresh all prices

**Primary actor:** Operator
**Supporting system:** PriceFollower API and collectors
**Trigger:** The operator chooses **Refresh prices** on the tracked-items page.

## Preconditions

- The tracked-items page has loaded.
- At least one item is tracked.
- No bulk refresh request is already being submitted by this page.

## Main flow

1. The page submits `POST /api/v1/items/refresh`.
2. The API accepts the request asynchronously and reports how many items are included.
3. The page shows refresh progress and reloads the list while item checks are pending.
4. The page stops showing progress when the included checks finish and displays the updated item data.

## Alternatives and errors

- Items that are already being checked are included in the result and are not checked a second time.
- If the request fails, the page shows an error and leaves existing price observations visible.
- If there are no tracked items, the refresh action is disabled.

## Postconditions

- Each included item has an immediate check queued or already in progress. The twice-daily schedule remains active.

# UC-ID-01 — View item details

**Primary actor:** Operator
**Supporting system:** PriceFollower API
**Trigger:** The operator chooses an item's title or details action, or opens `/items/{id}` directly.

## Preconditions

- The item exists in the shared tracked collection.

## Main flow

1. The page requests the item from `GET /api/v1/items/{id}`.
2. The page shows the item's title, platform, listing ID, source link, status, latest successful price, and its detection time.
3. The page shows all successful item-price observations, newest first.
4. For an Amazon item, the page shows the latest Amazon-sold second-hand offer and all successful qualifying offer observations with condition and time.
5. The page shows the next scheduled check when available and provides actions to refresh, open the source listing, delete the item, or return to the list.

## Alternatives and errors

- If the item has no successful price yet, the page explains that no price has been detected.
- If the initial check is pending, the page explains that the first price check is in progress and reloads the item while pending.
- A stale, retrieval-error, or unavailable status is shown while preserving the last successful price.
- If the item cannot be loaded, the page shows an error. If the item no longer exists, the API returns not found.
- LeBoncoin items have no Amazon second-hand offer section.

## Postconditions

- Viewing details does not change the item or its observations.

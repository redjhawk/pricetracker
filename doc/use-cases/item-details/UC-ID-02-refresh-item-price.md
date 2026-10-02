# UC-ID-02 — Refresh an item's price

**Primary actor:** Operator
**Supporting system:** PriceFollower API and platform collector
**Trigger:** The operator chooses **Refresh price** on an item's details page.

## Preconditions

- The item details page is open for an existing item.
- No refresh request is already being submitted from this page.

## Main flow

1. The page submits `POST /api/v1/items/{id}/refresh`.
2. The API accepts the regular item-price check asynchronously.
3. The page shows progress and reloads `GET /api/v1/items/{id}` while the item is pending.
4. When the check completes, the page displays the latest item state and any new observation.

## Alternatives and errors

- If a check is already running, the API does not start a duplicate; the page continues to show its result when available.
- If the refresh request fails, the page shows an error and keeps the last successful price visible.
- A failed collection attempt is reported without replacing the last successful price.

## Postconditions

- An immediate item-price check is queued or already in progress. The regular schedule remains active.

# UC-IM-01 — Add a listing

**Primary actor:** Operator
**Supporting system:** PriceFollower API and platform collectors
**Trigger:** The operator chooses **Add item** and submits a listing URL.

## Preconditions

- The operator has an Amazon listing from a supported European marketplace or a French LeBoncoin listing.

## Main flow

1. The operator enters the listing URL in the add-item form.
2. The page submits the URL to `POST /api/v1/items`.
3. The server validates the URL, supported platform, and listing form, then stores the item.
4. The server starts the first collection attempt immediately and returns the item without waiting for the marketplace request to finish.
5. The page adds the returned pending item to the tracked-items list and closes the form.
6. The first successful price observation appears when collection finishes.

## Alternatives and errors

- An empty URL is rejected by the form.
- A malformed URL, unsupported listing, or duplicate item produces an explanation in the form.
- If the first collection attempt fails, the saved item remains tracked and scheduled retries continue.
- The operator can cancel the form before submission without changing the collection.

## Postconditions

- On success, the item exists in the shared collection and its initial price check has started.

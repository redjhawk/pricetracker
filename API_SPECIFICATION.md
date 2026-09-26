# API Contract — Approved

**Status:** Initial contract approved 2026-09-24; Amazon second-hand offer, bulk refresh, LeBoncoin, and single-item refresh approved 2026-09-26.

## Conventions

- Base path: `/api/v1`
- JSON request and response bodies use UTF-8.
- Timestamps are ISO 8601 UTC strings.
- Prices are integer euro cents in the API (`amountCents`); `currency` is always `EUR` in v1.
- Supported v1 platforms are European Amazon marketplaces (`amazon.de`, `amazon.fr`, `amazon.es`, `amazon.it`, `amazon.nl`, `amazon.be`) and French LeBoncoin (`leboncoin.fr`). Amazon marketplaces whose normal listing prices are not in euros are excluded from v1.
- There is one shared item collection. No authentication is required.
- Item list order is most recently added first. The v1 collection is small (about 100 items), so list pagination is not required.

## Item response

```json
{
  "id": "item_123",
  "title": "Sony WH-1000XM5 Wireless Noise Canceling Headphones",
  "platform": "amazon",
  "listingId": "B09XS7JWHH",
  "asin": "B09XS7JWHH",
  "marketplace": "amazon.de",
  "url": "https://www.amazon.de/dp/B09XS7JWHH",
  "thumbnailUrl": null,
  "status": "active",
  "latestPrice": {
    "amountCents": 27900,
    "currency": "EUR",
    "timestamp": "2026-09-24T08:12:00Z"
  },
  "lastThreeDetections": [
    {
      "amountCents": 27900,
      "currency": "EUR",
      "timestamp": "2026-09-24T08:12:00Z"
    }
  ],
  "secondHandOffer": {
    "status": "available",
    "latestDetection": {
      "amountCents": 20900,
      "currency": "EUR",
      "condition": "good",
      "conditionLabel": "Bon état",
      "timestamp": "2026-09-24T08:12:00Z"
    },
    "lastThreeDetections": [
      {
        "amountCents": 20900,
        "currency": "EUR",
        "condition": "good",
        "conditionLabel": "Bon état",
        "timestamp": "2026-09-24T08:12:00Z"
      }
    ],
    "lastCheckedAt": "2026-09-24T08:12:00Z"
  },
  "lastAttempt": {
    "result": "success",
    "timestamp": "2026-09-24T08:12:00Z",
    "message": null
  },
  "nextCheckAt": "2026-09-24T20:00:00Z",
  "addedAt": "2026-09-01T14:30:00Z"
}
```

Fields:

- `title` and `thumbnailUrl` may be `null` if unavailable.
- `platform` is `amazon` or `leboncoin`; `listingId` is the source ASIN or numeric LeBoncoin ad ID. `asin` is the Amazon ASIN and is `null` for LeBoncoin items.
- `status` is one of `pending`, `active`, `stale`, `retrieval_error`, or `unavailable`.
- `latestPrice` is `null` until a successful price detection has been stored.
- `lastThreeDetections` contains up to three successful observations, newest first; it is empty if no price has been detected.
- `secondHandOffer.status` is `pending`, `available`, `not_found`, or `check_error`. Only offers explicitly sold by Amazon (including its identified Amazon Resale/Warehouse offers) qualify; all third-party offers are ignored. If seller attribution cannot be verified, the check is an error.
- `secondHandOffer.latestDetection` is the latest qualifying offer detection, even if a later check finds no offer or fails. The interface should use `status` to distinguish a current offer from an older last detection.
- `secondHandOffer.lastThreeDetections` contains up to three successful detections of the lowest-priced qualifying Amazon offer, newest first. Each detection includes normalized `condition` (`like_new`, `very_good`, `good`, `acceptable`, or `unknown`) and the original Amazon `conditionLabel`.
- `secondHandOffer.lastCheckedAt` is `null` before the first check.
- `secondHandOffer` is `null` for LeBoncoin items.
- A confirmed LeBoncoin free/donation listing has a successful observation with `amountCents: 0`; an absent or unparseable price is not interpreted as free.
- `lastAttempt` is `null` before the first collection attempt. Its `result` is one of `pending`, `success`, `request_error`, `price_not_found`, or `unavailable`. `message` is optional user-safe detail and must not expose internal errors.
- `nextCheckAt` may be `null` if a next attempt has not been scheduled yet.

## Endpoints

### List tracked items

`GET /api/v1/items`

Response `200 OK`:

```json
{ "items": [] }
```

Each element is an item response. An empty collection returns an empty array.

### Refresh all tracked prices

`POST /api/v1/items/refresh`

No request body is required. The backend schedules an immediate check for each item in the collection at the time of the request. Each check includes both the regular item price and the lowest-priced qualifying Amazon-sold second-hand offer. Items already being checked count as included and are not checked a second time. The existing twice-daily schedule remains active.

Response `202 Accepted`:

```json
{
  "requestedAt": "2026-09-26T10:00:00Z",
  "itemsQueued": 12
}
```

The existing item list reports queued or running collection attempts as `pending`, allowing the frontend to poll `GET /api/v1/items` until checks finish. `itemsQueued` is the number of tracked items included, including those already being checked.

### Refresh one item

`POST /api/v1/items/{id}/refresh`

No request body is required. The backend schedules an immediate check of the specified item's regular price. A check already running is not duplicated. The normal twice-daily schedule remains active.

Response `202 Accepted`:

```json
{
  "requestedAt": "2026-09-26T10:00:00Z",
  "itemsQueued": 1
}
```

A missing item returns `404 ITEM_NOT_FOUND`. The detail page polls `GET /api/v1/items/{id}` while the check is pending.

### Get item details

`GET /api/v1/items/{id}`

Response `200 OK`: one item response. A missing item returns `404 Not Found`.

### Add an item

`POST /api/v1/items`

Request:

```json
{ "url": "https://www.amazon.de/dp/B09XS7JWHH" }
```

The backend accepts URLs from the configured European Amazon marketplaces or a LeBoncoin ad matching `https://www.leboncoin.fr/ad/{category}/{numericListingId}`. Tracking query parameters are removed when canonicalizing; one final comma after the numeric listing ID is accepted as pasted punctuation. A duplicate canonical listing returns `409 Conflict`. Invalid JSON or a missing/malformed URL returns `400 Bad Request`; a well-formed URL from an unsupported marketplace or listing form returns `422 Unprocessable Content`.

Response `201 Created`: the new item response. The item is stored immediately with `status: "pending"`; the backend starts its initial price collection attempt immediately without making the client wait for Amazon. The normal twice-daily schedule follows. The frontend can refresh the item list or detail to display the first result when it arrives.

### Delete an item

`DELETE /api/v1/items/{id}`

Response `204 No Content` when the item and its observations are deleted. A missing item returns `404 Not Found`. Deletion removes all stored price history for the item.

## Error response

Errors use a consistent JSON shape:

```json
{
  "error": {
    "code": "ITEM_ALREADY_TRACKED",
    "message": "This listing is already being tracked."
  }
}
```

The client may use `code` for predictable UI behavior. `message` is safe to display. Internal stack traces, upstream response bodies, and sensitive request data are never returned.

## Frontend behavior supported by this contract

- The list view loads all current items with one request and can filter them locally by title, listing ID, platform, and marketplace.
- The main list shows the latest qualifying Amazon-sold second-hand offer and its Amazon condition for Amazon items. It distinguishes an available offer, no qualifying offer, a pending check, and a failed/uncertain check. LeBoncoin items have no second-hand-offer data.
- The details view can show the last three second-hand offer detections and their condition labels.
- The main page can request a refresh of all tracked items and displays progress while the existing item statuses are pending.
- The details view loads a single item by ID and can render its latest price, recent detections, and collection status.
- The details view can request an immediate check of its item and reloads server data while that check is pending.
- Adding an item shows validation/duplicate errors from the API and displays the newly created pending item while its immediate collection attempt runs.
- Deleting an item removes it from the visible list after a successful response.
- Stale or failed attempts do not remove or overwrite the last successful price.

## LeBoncoin and per-item refresh

This API extension was approved on 2026-09-26.

### Item response changes

Add a `platform` field with values `amazon` or `leboncoin`, and a `listingId` field containing the platform's identifier (Amazon ASIN or numeric LeBoncoin ad ID). Keep `asin` for compatibility with Amazon clients; it remains the ASIN for Amazon items and is `null` for LeBoncoin items. `marketplace` remains the host, using values such as `amazon.fr` and `leboncoin.fr`.

For Amazon items, `secondHandOffer` retains its existing object shape. For LeBoncoin items, `secondHandOffer` is `null`; its existing meaning is specific to Amazon-sold offers.

Successful price observations retain their existing shape and euro-cent representation. A confirmed free/donation listing is represented as `amountCents: 0` and displayed as “Gratuit”. An unparseable or absent price is not treated as zero.

### Add a LeBoncoin listing

Keep `POST /api/v1/items` and its `{ "url": "..." }` request shape. In addition to the supported Amazon URLs, accept LeBoncoin listing URLs matching:

```text
https://www.leboncoin.fr/ad/{category}/{numericListingId}
```

Canonicalize tracking query parameters away. Accept a trailing comma after the numeric ID as pasted punctuation, so the two example links are accepted. Reject other LeBoncoin pages and malformed IDs with `422 UNSUPPORTED_LISTING`. A duplicate canonical URL returns the existing `409 ITEM_ALREADY_TRACKED` response. A valid listing is returned immediately with `status: "pending"` and is collected asynchronously, as for Amazon.

LeBoncoin items use the same `latestPrice`, `lastThreeDetections`, attempt status, deletion, and twice-daily scheduling behavior as Amazon items. A collection failure or unavailable listing does not replace the last successful price. Explicit free/donation listings record a zero-euro observation; missing or unparseable prices report `price_not_found` rather than a free price.

### Refresh one item

Add:

```text
POST /api/v1/items/{id}/refresh
```

No body is required. Queue an immediate collection for that item, including its regular item price. The existing scheduled checks remain active. If a check for the item is already running, do not start a duplicate; return `202 Accepted` with the same response shape.

Response `202 Accepted`:

```json
{
  "requestedAt": "2026-09-26T10:00:00Z",
  "itemsQueued": 1
}
```

A missing item returns `404 ITEM_NOT_FOUND`. The detail page displays progress while the item's status is `pending`, then reloads the server item details to show the latest result.

### Required frontend behavior

- Show whether each item is from Amazon or LeBoncoin; retain Amazon second-hand offer details only for Amazon items.
- Keep price history and regular price display consistent across both platforms, including “Gratuit” for a confirmed zero-euro listing.
- Add a refresh button to the item details view. It calls the single-item endpoint, shows pending feedback, and reloads the item until collection completes.

### LeBoncoin page data

The supplied listings return HTML with a Next.js `__NEXT_DATA__` script containing `props.pageProps.ad`. Its `list_id`, `status`, `subject`, `body`, `price_cents`, and image data provide the listing identifier, availability, title, description, euro price, and thumbnail. Explicit donation text is used only when the ad has no price field; that case records a zero-euro observation. Other missing prices remain `price_not_found`. Requests must include the configured Chrome browser headers; a DataDome challenge or other non-success HTTP response is a `request_error`, not an unavailable listing or a free price.

Only euro-priced marketplaces are supported in v1: `amazon.de`, `amazon.fr`, `amazon.es`, `amazon.it`, `amazon.nl`, `amazon.be`, and `leboncoin.fr`. Listings from marketplaces whose displayed prices are in another currency are rejected rather than converted.

## Confirmed API decisions

1. Endpoint paths and methods are approved.
2. Integer euro cents are approved as the API price representation.
3. The status and last-attempt result values are approved.
4. `201 Created` is returned immediately while the first collection attempt runs asynchronously.
5. The list returns all tracked items in one response for the initial limit of around 100.

# API Contract — Draft for Approval

**Status:** Initial contract approved 2026-09-24; Amazon second-hand offer and bulk refresh additions approved 2026-09-26.

## Conventions

- Base path: `/api/v1`
- JSON request and response bodies use UTF-8.
- Timestamps are ISO 8601 UTC strings.
- Prices are integer euro cents in the API (`amountCents`); `currency` is always `EUR` in v1.
- Supported v1 Amazon marketplaces are `amazon.de`, `amazon.fr`, `amazon.es`, `amazon.it`, `amazon.nl`, and `amazon.be`. Amazon marketplaces whose normal listing prices are not in euros are excluded from v1.
- There is one shared item collection. No authentication is required.
- Item list order is most recently added first. The v1 collection is small (about 100 items), so list pagination is not required.

## Item response

```json
{
  "id": "item_123",
  "title": "Sony WH-1000XM5 Wireless Noise Canceling Headphones",
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
- `status` is one of `pending`, `active`, `stale`, `retrieval_error`, or `unavailable`.
- `latestPrice` is `null` until a successful price detection has been stored.
- `lastThreeDetections` contains up to three successful observations, newest first; it is empty if no price has been detected.
- `secondHandOffer.status` is `pending`, `available`, `not_found`, or `check_error`. Only offers explicitly sold by Amazon (including its identified Amazon Resale/Warehouse offers) qualify; all third-party offers are ignored. If seller attribution cannot be verified, the check is an error.
- `secondHandOffer.latestDetection` is the latest qualifying offer detection, even if a later check finds no offer or fails. The interface should use `status` to distinguish a current offer from an older last detection.
- `secondHandOffer.lastThreeDetections` contains up to three successful detections of the lowest-priced qualifying Amazon offer, newest first. Each detection includes normalized `condition` (`like_new`, `very_good`, `good`, `acceptable`, or `unknown`) and the original Amazon `conditionLabel`.
- `secondHandOffer.lastCheckedAt` is `null` before the first check.
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

### Get item details

`GET /api/v1/items/{id}`

Response `200 OK`: one item response. A missing item returns `404 Not Found`.

### Add an item

`POST /api/v1/items`

Request:

```json
{ "url": "https://www.amazon.de/dp/B09XS7JWHH" }
```

The backend validates the URL and restricts it to the configured European Amazon marketplaces. A duplicate canonical listing returns `409 Conflict`. Invalid JSON or a missing/malformed URL returns `400 Bad Request`; a well-formed URL from an unsupported marketplace or listing form returns `422 Unprocessable Content`.

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

- The list view loads all current items with one request and can filter them locally by title, ASIN, and marketplace.
- The main list shows the latest qualifying Amazon-sold second-hand offer and its Amazon condition. It distinguishes an available offer, no qualifying offer, a pending check, and a failed/uncertain check.
- The details view can show the last three second-hand offer detections and their condition labels.
- The main page can request a refresh of all tracked items and displays progress while the existing item statuses are pending.
- The details view loads a single item by ID and can render its latest price, recent detections, and collection status.
- Adding an item shows validation/duplicate errors from the API and displays the newly created pending item while its immediate collection attempt runs.
- Deleting an item removes it from the visible list after a successful response.
- Stale or failed attempts do not remove or overwrite the last successful price.

Only euro-priced marketplaces are supported in v1: `amazon.de`, `amazon.fr`, `amazon.es`, `amazon.it`, `amazon.nl`, and `amazon.be`. Listings from marketplaces whose displayed prices are in another currency are rejected rather than converted.

## Decisions to confirm

1. Endpoint paths and methods are approved.
2. Integer euro cents are approved as the API price representation.
3. The status and last-attempt result values are approved.
4. `201 Created` is returned immediately while the first collection attempt runs asynchronously.
5. The list returns all tracked items in one response for the initial limit of around 100.

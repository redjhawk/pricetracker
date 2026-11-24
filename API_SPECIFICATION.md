# API Contract — Approved

**Status:** Initial contract approved 2026-09-24; Amazon second-hand offer, bulk refresh, LeBoncoin, and single-item refresh approved 2026-09-26; full item price history approved 2026-10-01; consecutive item price periods approved 2026-10-02; LeBoncoin session settings approved 2026-10-03; Claude token settings and AI reviews defined 2026-10-04 (agent-defined per AGENTS.md); LeBoncoin purchase goal defined 2026-10-04 (agent-defined); users and login phase 1 defined 2026-10-04 (agent-defined); Amazon searches defined 2026-10-07 (agent-defined, issue #56); session `version` field defined 2026-11-10 (agent-defined, issue #115); Amazon human-like checks behaviour defined 2026-11-20 (agent-defined, issue #168, no shape change); Amazon success response not opened as a page defined 2026-11-22 (agent-defined, issue #186, no shape change); Amazon unavailable listing on the browser's error page defined 2026-11-23 (agent-defined, issue #187, no shape change); masked Settings secrets defined 2026-11-23 (agent-defined, issue #265: settings objects return `saved`/`hint` instead of `value`, `PUT /api/v1/settings` gains explicit removal and treats an empty value as unchanged, `PUT /api/v1/settings/leboncoin-session` removed); Amazon thumbnail URLs restricted to Amazon image hosts over HTTPS defined 2026-11-24 (agent-defined, issue #275, no shape change); HTTP server read, write and header-size limits defined 2026-11-24 (agent-defined, issue #273, no shape change).

## Conventions

- Base path: `/api/v1`
- JSON request and response bodies use UTF-8.
- Timestamps are ISO 8601 UTC strings.
- Prices are integer euro cents in the API (`amountCents`); `currency` is always `EUR` in v1.
- Supported v1 platforms are European Amazon marketplaces (`amazon.de`, `amazon.fr`, `amazon.es`, `amazon.it`, `amazon.nl`, `amazon.be`) and French LeBoncoin (`leboncoin.fr`). Amazon marketplaces whose normal listing prices are not in euros are excluded from v1.
- Without regular users (open mode) there is one item collection and no authentication is required. Once a regular user exists (protected mode), a session is required and items and settings are scoped to the logged-in user; see "Users and login".
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
- An Amazon check records as `thumbnailUrl` only the first product page image URL starting exactly with `https://m.media-amazon.com/` or `https://images-eu.ssl-images-amazon.com/` (main image, then the first dynamic-size image, then `og:image`), kept as found. When the page offers none, the check records no thumbnail: a new item keeps `null` and an existing thumbnail is kept. Values stored before issue #275 are not rewritten. See [Amazon thumbnail URLs](doc/specifications/amazon-thumbnail-urls/technical.md).
- `platform` is `amazon` or `leboncoin`; `listingId` is the source ASIN or numeric LeBoncoin ad ID. `asin` is the Amazon ASIN and is `null` for LeBoncoin items.
- `status` is one of `pending`, `active`, `stale`, `retrieval_error`, or `unavailable`.
- `latestPrice` is `null` until a successful price detection has been stored.
- `lastThreeDetections` contains up to three recent consecutive item-price periods, newest first. Each entry contains the period's price and timestamp of its latest successful observation. Repeated observations at the same price update that period's timestamp; a different price starts a new period. A later return to an earlier price starts another period. It is empty if no price has been detected.
- `priceHistory` is included by `GET /api/v1/items/{id}` and contains every successful item-price observation for the tracked item, newest first. It is omitted by the list and add responses.
- `secondHandOffer.status` is `pending`, `available`, `not_found`, or `check_error`. Only offers explicitly sold by Amazon (including its identified Amazon Resale/Warehouse offers) qualify; all third-party offers are ignored. If seller attribution cannot be verified, the check is an error.
- `secondHandOffer.latestDetection` is the latest qualifying offer detection, even if a later check finds no offer or fails. The interface should use `status` to distinguish a current offer from an older last detection.
- `secondHandOffer.lastThreeDetections` contains up to three successful detections of the lowest-priced qualifying Amazon offer, newest first. Each detection includes normalized `condition` (`like_new`, `very_good`, `good`, `acceptable`, or `unknown`) and the original Amazon `conditionLabel`.
- `secondHandOffer.priceHistory` is included in `GET /api/v1/items/{id}` and contains every successful qualifying Amazon-sold offer detection, newest first. It is omitted by the list and add responses. It is `null` for LeBoncoin because `secondHandOffer` is `null` there.
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
The detail response includes complete `priceHistory` arrays for item prices and, when applicable, Amazon second-hand offers. Histories are unpaginated and include all successful observations retained for that tracked item. Failed collection attempts do not create price observations. The list and add responses return up to three recent consecutive item-price periods in `lastThreeDetections`; the timestamp for each period is its latest successful observation. Amazon second-hand offer detection behavior is unchanged.

### Add an item

`POST /api/v1/items`

Request:

```json
{ "url": "https://www.amazon.de/dp/B09XS7JWHH" }
```

The backend accepts URLs from the configured European Amazon marketplaces or a LeBoncoin ad matching `https://www.leboncoin.fr/ad/{category}/{numericListingId}`. Amazon's `url` field retains the submitted listing URL and collection requests use that URL; URL normalization is used separately for duplicate detection. LeBoncoin tracking query parameters are removed when canonicalizing; one final comma after the numeric listing ID is accepted as pasted punctuation. A duplicate canonical listing returns `409 Conflict`. Invalid JSON or a missing/malformed URL returns `400 Bad Request`; a well-formed URL from an unsupported marketplace or listing form returns `422 Unprocessable Content`.

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
- The details view displays every successful item-price detection and every successful Amazon-sold second-hand offer detection with its condition label.
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

## LeBoncoin session settings (approved 2026-10-03)

**Status: approved by the user on 2026-10-03** (proposal revision 1, unchanged) for change [leboncoin-session-settings](doc/changes/2026-10-03-1741-leboncoin-session-settings/index.md); approval evidence in its [API record](doc/changes/2026-10-03-1741-leboncoin-session-settings/api-step.md). Technical sources: [settings](doc/specifications/leboncoin-session-settings/technical.md), [collection rev. 2](doc/specifications/leboncoin-session-collection/technical.md).

The application stores one LeBoncoin `datadome` session value per settings owner in SQLite (see "Users and login"). Since issue #265 the saved value never leaves the server: responses state whether a value is saved and give only its hint (see "Secret hint"). Responses carry the existing `Cache-Control: no-store`.

### Secret hint (issue #265)

The saved LeBoncoin session value and the saved Claude token are never returned, in any response. Each settings object instead carries:

| Field | Type | Meaning |
| --- | --- | --- |
| `saved` | boolean | `true` when a value is stored (whatever its `status`). |
| `hint` | string or `null` | The last 4 characters of the stored value when it is at least 16 characters long; `null` when nothing is saved or the value is shorter than 16 characters. Never more than 4 characters. It is display text only: clients must never send it back as a value. |

### Session settings object

```json
{
  "session": {
    "saved": true,
    "hint": "mple",
    "revision": 7,
    "updatedAt": "2026-10-03T18:00:00.000Z",
    "status": "active",
    "expiresAt": "2027-10-03T18:00:00.000Z",
    "revokedAt": null,
    "lastAttempt": {
      "outcome": "rejected",
      "attemptedAt": "2026-10-03T20:00:00.000Z"
    }
  }
}
```

| Field | Type | Meaning |
| --- | --- | --- |
| `saved`, `hint` | boolean, string or `null` | See "Secret hint". The stored `datadome` value (1–4096 characters) itself is never returned (issue #265; it replaced the former `value` field). |
| `revision` | integer ≥ 0 | Change token. Increases when the operator replaces or removes the session, when collection stores a renewed value, and when LeBoncoin revokes the session. Must be sent back when replacing or removing the session. |
| `updatedAt` | string or `null` | Time of the last stored change; `null` if never set. |
| `status` | `none` \| `active` \| `expired` \| `revoked` | `none`: no value. `active`: value used for LeBoncoin checks. `expired`: a known expiry supplied by LeBoncoin has passed. `revoked`: LeBoncoin deleted the cookie. `expired`/`revoked` values are kept (and shown by their hint) but not used; LeBoncoin checks record `request_error` until a new value is saved or the session is removed. |
| `expiresAt` | string or `null` | Known expiry from a LeBoncoin renewal; `null` for a pasted value or a session cookie. |
| `revokedAt` | string or `null` | When LeBoncoin revoked the value; non-null only with `status: "revoked"`. |
| `lastAttempt` | object or `null` | Latest LeBoncoin check that sent this session (since it was last saved by the operator); `null` when unused. `outcome`: `accepted` (LeBoncoin returned the requested listing), `rejected` (HTTP 403 or verification challenge), `failed` (any other failure, including network errors and unavailable listings). `attemptedAt`: UTC time. |

The UI shows a warning for `status` `expired`/`revoked`, otherwise for `lastAttempt.outcome` `rejected` or `failed`; no warning for `none`, `null` or `accepted`.

### Get the LeBoncoin session

```text
GET /api/v1/settings/leboncoin-session
```

`200 OK` with the session settings object. With no saved session: `{"session": {"saved": false, "hint": null, "revision": 0, "updatedAt": null, "status": "none", "expiresAt": null, "revokedAt": null, "lastAttempt": null}}` (revision may be higher if a session was saved and removed before). `405 METHOD_NOT_ALLOWED` for any other method (`Allow: GET`). `500 INTERNAL_ERROR` if it cannot be read.

### LeBoncoin session input (used by `PUT /api/v1/settings`)

`PUT /api/v1/settings/leboncoin-session` was removed by issue #265 (it returned the full value and cleared the session on an empty value); the path now answers `405 METHOD_NOT_ALLOWED` with `Allow: GET` to `PUT`. The session is saved only through `PUT /api/v1/settings`, whose `leboncoinSession.value` follows these rules:

- The server trims surrounding whitespace and accepts either the raw value or a cookie string (optionally prefixed with `Cookie:` or `Set-Cookie:`, case-insensitive) from which it extracts the `datadome` pair; other pairs and attributes are discarded and never stored. An empty or whitespace-only string leaves the session unchanged (issue #265), whatever its length: a whitespace-only string is never rejected as too long (issue #286).
- `400 INVALID_SESSION` when the input is unusable: longer than 8192 characters and not whitespace only; cookie string without `datadome`; empty `datadome` value; several different `datadome` values; value with whitespace, control characters, non-ASCII characters, `;`, `,`, `"`, `\` or other invalid cookie characters; value longer than 4096 characters. Example message: “No datadome cookie was found in the pasted text.” Messages never repeat the input.
- A replaced session has `status: "active"`, `expiresAt: null`, `revokedAt: null`, `lastAttempt: null` and an incremented `revision`; a removed session has `saved: false`, `hint: null`, `status: "none"`. Saving never starts a price check; the next LeBoncoin check (scheduled or requested) uses the new state.

### Effect on existing endpoints

No item endpoint, field, status or result value changes. For LeBoncoin items checked with a saved session, `lastAttempt.message` (free text) may read “LeBoncoin rejected the saved session. Capture a new session and save it in Settings.” or “The saved LeBoncoin session has expired or was revoked. Save a new session in Settings.” or “The LeBoncoin session could not be read. The check will be retried at the next scheduled time.” with `result: "request_error"`. The session value never appears in item responses or messages.

## Claude token settings and AI reviews (2026-10-04)

**Status: defined by the technical specification agent on 2026-10-04** for change [leboncoin-ai-review](doc/changes/2026-10-04-0710-issue-4-leboncoin-ai-review/index.md) (no user confirmation required, AGENTS.md). Technical sources: [Claude token settings](doc/specifications/claude-token-settings/technical.md), [LeBoncoin AI review](doc/specifications/leboncoin-ai-review/technical.md). Additive: all earlier contracts above are unchanged. Responses carry the existing `Cache-Control: no-store`.

### Claude token object

```json
{ "claudeToken": { "saved": true, "hint": "mple", "updatedAt": "2026-10-04T09:00:00.000Z", "lastRejectedAt": null } }
```

| Field | Type | Meaning |
| --- | --- | --- |
| `saved`, `hint` | boolean, string or `null` | See "Secret hint" (issue #265). The stored Claude subscription token (1–1024 characters) itself is never returned; it replaced the former `value` field. |
| `updatedAt` | string or `null` | Last replacement/removal time. |
| `lastRejectedAt` | string or `null` | Time of the latest AI review attempt where Claude rejected this token (HTTP 401/403) or reported a usage limit (429). Cleared by any save and by a later successful review. The UI shows a replace-token warning when non-null. |

### Get the Claude token

```text
GET /api/v1/settings/claude-token
```

`200 OK` with the Claude token object (`{"claudeToken": {"saved": false, "hint": null, "updatedAt": null, "lastRejectedAt": null}}` when never set). `405` for other methods (`Allow: GET`). `500 INTERNAL_ERROR` on storage failure.

### Save settings (both Settings entries, atomically)

```text
PUT /api/v1/settings
```

Request (JSON, at most 32 768 bytes). Each part is optional; an absent part leaves that secret exactly as it is (issue #265). A present part either replaces (`value`) or removes (`"remove": true`) its secret:

```json
{
  "leboncoinSession": { "value": "datadome=Xyz~AbC123_example", "revision": 7 },
  "claudeToken": { "remove": true }
}
```

- `leboncoinSession` (optional): `revision` (integer ≥ 0, required) is the `revision` received when Settings was loaded, plus exactly one of:
  - `value` (string): the input as typed, under the rules of "LeBoncoin session input". A non-empty value replaces the session, even when it equals the stored one (new `revision`, warnings cleared). An empty or whitespace-only value, of any length, leaves the session unchanged and is not checked against `revision` (issue #286).
  - `remove` (`true`): removes the session (no change when none is saved).
- `claudeToken` (optional), exactly one of:
  - `value` (string): the input as typed. The server trims surrounding whitespace. A non-empty value is validated and always verified with a minimal Claude request before storing, even when it equals the stored token; storing clears `lastRejectedAt`. An empty or whitespace-only value, of any length, leaves the token unchanged and makes no verification; the 1024-character limit applies only to a value that is not whitespace only (issue #286).
  - `remove` (`true`): removes the token without verification (no change when none is saved).
- An untouched secret (part absent, or empty or whitespace-only `value`) keeps its value, `revision`, `updatedAt` and warnings (`lastRejectedAt`; session `status`, `expiresAt`, `revokedAt`, `lastAttempt`). The LeBoncoin `revision` is checked only when the session is replaced or removed.
- `{}`, or both parts untouched, is valid: nothing changes and the current objects are returned.
- Validation order: request shape, LeBoncoin input, Claude token format, Claude verification, then one transaction (LeBoncoin revision check and both writes). The first error is returned and **nothing** is saved. Error messages never repeat the submitted input.

Success `200 OK`: `{ "session": <session settings object>, "claudeToken": <Claude token object content> }`, both masked (`saved`/`hint`). Saving never starts a price check or an AI review.

| Status | `code` | When | Example `message` (safe to display) |
| --- | --- | --- | --- |
| 400 | `INVALID_JSON` | Body is not one JSON value | "Request body must be valid JSON." |
| 400 | `INVALID_REQUEST` | A part that is not an object; with neither or both of `value` and `remove`; with `remove` other than `true`; with a wrongly typed field; or a `leboncoinSession` without a non-negative integer `revision` | "Each Settings entry sent must have either a value or \"remove\": true, and the LeBoncoin session needs the revision from Settings." |
| 400 | `INVALID_SESSION` | As in "LeBoncoin session input" | "No datadome cookie was found in the pasted text." |
| 400 | `INVALID_CLAUDE_TOKEN` | Token contains whitespace, control or non-ASCII characters, or exceeds 1024 characters | "The Claude token must not contain spaces, line breaks or other control characters." |
| 422 | `CLAUDE_TOKEN_REJECTED` | Claude answered 401/403 to the verification request | "Claude refused this token. Create a new one with claude setup-token." |
| 502 | `CLAUDE_UNREACHABLE` | Network error, timeout (20 s), 429, 5xx or other unexpected answer during verification | "The token could not be verified because Claude could not be reached. Try again." |
| 409 | `SESSION_CHANGED` | The session is replaced or removed and the LeBoncoin `revision` is stale (another save or removal, automatic renewal, or revocation since loading) | "The LeBoncoin session changed after Settings was opened. Reopen Settings before saving." |
| 413 | `REQUEST_TOO_LARGE` | Body over 32 768 bytes | "Request body is too large." |
| 405 | `METHOD_NOT_ALLOWED` | Method other than PUT (`Allow: PUT`) | "This method is not allowed for the route." |
| 500 | `INTERNAL_ERROR` | Storage failure; nothing partially saved | "The server could not complete the request." |

Compatibility (issue #265): both settings objects return `saved`/`hint` instead of `value`; an empty `value` no longer removes a secret; `{}` is no longer `INVALID_REQUEST`; an equal Claude token is verified again; `PUT /api/v1/settings/leboncoin-session` is removed. The embedded frontend changes in the same release; there is no other client.

### AI review in the item details response

`GET /api/v1/items/{id}` adds `aiReview`: an object for LeBoncoin items, `null` for Amazon items. `GET /api/v1/items` (list) does not include it. All other item fields are unchanged.

```json
"aiReview": {
  "tokenConfigured": true,
  "running": false,
  "lastAttempt": { "id": 12, "status": "failed", "priceCents": null, "createdAt": "2026-10-04T10:00:00.000Z", "completedAt": "2026-10-04T10:00:20.000Z", "review": null, "errorMessage": "Claude could not be reached. Try again later." },
  "latest": { "id": 11, "status": "succeeded", "priceCents": 15000, "createdAt": "2026-10-03T10:00:00.000Z", "completedAt": "2026-10-03T10:00:40.000Z", "errorMessage": null, "review": {
    "price": { "rating": "fair", "explanation": "…" },
    "condition": { "rating": "good", "explanation": "…" },
    "recommendation": { "rating": "negotiate", "explanation": "…" },
    "fairPrice": { "minCents": 12000, "maxCents": 16000, "suggestedOfferCents": 13000 },
    "risks": [],
    "missingInformation": ["Charger not shown"],
    "sellerQuestions": ["Is the original charger included?"],
    "descriptionVsPhotos": { "matches": true, "explanation": "…", "mismatches": [] }
  } },
  "history": [ "…succeeded review objects, same shape as latest…" ]
}
```

| Field | Type | Meaning |
| --- | --- | --- |
| `tokenConfigured` | boolean | A Claude token is saved. When false, no review is started and the refresh action is unavailable. |
| `running` | boolean | A review for this item is in progress (at most one). Clients poll the details endpoint (e.g. every 3 s) while true. |
| `lastAttempt` | review or `null` | Most recent review attempt of any status (`pending` while running). |
| `latest` | review or `null` | Most recent `succeeded` review; `null` if none ("no review yet"). |
| `history` | review[] | `succeeded` reviews, newest first (includes `latest`), at most 50. |

Review object: `id` (integer); `status` (`pending`, `succeeded` or `failed`); `priceCents` (integer euro cents the review was based on, `0` = free, `null` when unknown or not succeeded); `createdAt`; `completedAt` (string, `null` while pending); `review` (object when `succeeded`, else `null`); `errorMessage` (safe English text when `failed`, else `null`; never contains the token).

`review` fields: `price.rating` one of `good_deal`, `fair`, `overpriced`; `condition.rating` one of `excellent`, `good`, `fair`, `poor`, or `null` when it cannot be assessed from photos; `recommendation.rating` one of `buy`, `negotiate`, `avoid`; each with a non-empty English `explanation`. `fairPrice`: `minCents ≤ maxCents` and `suggestedOfferCents`, integers ≥ 0. `risks` (empty = no risk signs found), `missingInformation`, `sellerQuestions`: string arrays. `descriptionVsPhotos`: `matches` boolean or `null` (cannot compare), `explanation` string, `mismatches` string array.

Failure messages: "Claude rejected the token. Replace it in Settings.", "Claude usage limit reached. Try again later.", "Claude could not be reached. Try again later.", "Claude returned an unusable review. Try again.", "The listing could not be retrieved from LeBoncoin.", "The review was interrupted by a server restart."

Automatic reviews (server-side, asynchronous): after the first collection of a newly added LeBoncoin item, and after any collection whose recorded price differs from the previous recorded price; only when a token is saved and no review is running for the item.

### Request an AI review

```text
POST /api/v1/items/{id}/ai-review
```

No body. Fetches the current listing from LeBoncoin and reviews it asynchronously; does not record a price.

`202 Accepted`: `{ "requestedAt": "2026-10-04T10:00:00.000Z", "alreadyRunning": false }`. If a review is already running, no new one starts and `alreadyRunning` is `true`.

| Status | `code` | When | Example `message` |
| --- | --- | --- | --- |
| 404 | `ITEM_NOT_FOUND` | Unknown item | "Tracked item was not found." |
| 409 | `CLAUDE_TOKEN_MISSING` | No Claude token saved | "Configure a Claude token in Settings." |
| 422 | `AI_REVIEW_UNSUPPORTED` | Amazon item | "AI reviews are available for LeBoncoin items only." |
| 405 | `METHOD_NOT_ALLOWED` | Method other than POST (`Allow: POST`) | "This method is not allowed for the route." |
| 500 | `INTERNAL_ERROR` | Storage failure | "The server could not complete the request." |

Deleting an item deletes its reviews.

## LeBoncoin purchase goal (2026-10-04)

Agent-defined per AGENTS.md for [LeBoncoin purchase goal](doc/specifications/leboncoin-purchase-goal/technical.md). Rationale: the operator states a free-text purchase goal per LeBoncoin item that is included in every AI review; editing it must start a new review. Additive and backward compatible.

### Item response change

All item responses (list, add, details) add `purchaseGoal`: string, `""` when no goal is set; always `""` for Amazon items. Leading/trailing whitespace is trimmed; no maximum length is enforced. Review objects do not include the goal.

### Add an item change

`POST /api/v1/items` accepts an optional `purchaseGoal` string: `{ "url": "https://www.leboncoin.fr/ad/...", "purchaseGoal": "Install a light Linux distro" }`. It is trimmed and stored for LeBoncoin items and ignored for Amazon items. A non-string value returns `400 INVALID_JSON`. The request body limit is 1 MiB (`413 REQUEST_TOO_LARGE` above it). Other behavior is unchanged.

### Update the purchase goal

```text
PUT /api/v1/items/{id}/purchase-goal
```

Request: `{ "purchaseGoal": "Install a light Linux distro" }` (`""` clears the goal; body limit 1 MiB).

`200 OK`: `{ "purchaseGoal": "Install a light Linux distro", "changed": true, "reviewStarted": true }`.

- `purchaseGoal`: the stored (trimmed) value.
- `changed`: `false` when the trimmed value equals the stored one; nothing is written and no review starts.
- `reviewStarted`: `true` when a change requested a new AI review (asynchronous; fetches the listing like `POST /ai-review`). If a review is already running, one more review runs after it finishes, using the newest goal. `false` when unchanged or no Claude token is saved.

| Status | `code` | When | Example `message` |
| --- | --- | --- | --- |
| 400 | `INVALID_JSON` | Invalid JSON, more than one value, or `purchaseGoal` missing/not a string | "Request body must be valid JSON." |
| 404 | `ITEM_NOT_FOUND` | Unknown item | "Tracked item was not found." |
| 405 | `METHOD_NOT_ALLOWED` | Method other than PUT (`Allow: PUT`) | "This method is not allowed for the route." |
| 413 | `REQUEST_TOO_LARGE` | Body over 1 MiB | "Request body is too large." |
| 422 | `PURCHASE_GOAL_UNSUPPORTED` | Amazon item | "Purchase goals are available for LeBoncoin items only." |
| 500 | `INTERNAL_ERROR` | Storage failure | "The server could not complete the request." |

### Effect on AI reviews

Every review (automatic, manual, or goal change) reads the item's goal when the review runs. A non-empty goal is sent to Claude as the buyer's purchase goal, delimited as untrusted data; an empty goal leaves the request unchanged. The review structure is unchanged.

## Users and login (phase 1, 2026-10-04)

Agent-defined from [the technical specification](doc/specifications/multi-user-login/technical.md). Existing request and response shapes are unchanged.

### Modes, session and access

- Mode is `open` while no regular user exists, `protected` from the first created user onwards.
- Session: HttpOnly cookie `pricefollower_session` (`SameSite=Lax`, `Path=/`), set by login, valid 30 days from login, not extended, ended by logout.
- Since 2026-11-24 (issue #268, [technical spec](doc/specifications/session-cookie-security/technical.md)): the cookie set by login and the clearing cookie of logout carry `Secure` when the request is HTTPS: received over TLS by the server, or carrying an `X-Forwarded-Proto` header whose first header's first comma-separated entry is `https` (case-insensitive, surrounding spaces ignored). Otherwise no `Secure`, so plain HTTP in development keeps working. The header is trusted because the application is reachable only through Cloudflare. No other cookie attribute, request or response changes.
- `/api/v1/auth/*` is always reachable.
- A logged-in `admin` may call only `/api/v1/auth/*` and `/api/v1/admin/*`; any other endpoint returns `403 FORBIDDEN`, in both modes.
- `/api/v1/admin/*` returns `401 AUTH_REQUIRED` without an admin session and `403 FORBIDDEN` for a regular user.
- Protected mode: every other endpoint returns `401 AUTH_REQUIRED` without a valid session. A regular user sees and changes only their own items, price history, purchase goals, AI reviews and settings (Claude token, LeBoncoin session). An item owned by someone else returns `404 ITEM_NOT_FOUND`, like an unknown id. `ITEM_ALREADY_TRACKED` applies only to the same user's items; two users may track the same URL independently.
- Open mode without a session: all existing endpoints behave as before, on the open-mode items and settings. When the first regular user is created, these become theirs.

| Status | Code | When | Message |
| --- | --- | --- | --- |
| 401 | `AUTH_REQUIRED` | Session missing, expired or ended where one is required | "Log in to continue." |
| 401 | `INVALID_CREDENTIALS` | Wrong username or password | "Incorrect username or password." |
| 403 | `FORBIDDEN` | Role not allowed for the endpoint | "You do not have access to this page." |
| 429 | `LOGIN_LOCKED` | 5 consecutive failed logins for the client address and username (case-insensitive); that pair is refused for 1 minute without checking the password (issue #266) | "Too many failed attempts. Wait one minute and try again." |
| 429 | `LOGIN_BUSY` | Login only: the maximum number of password checks (4) are already waiting; answered at once, not counted as a failure (issue #266) | "Too many login attempts are being checked. Try again in a moment." |

### Get the session state

`GET /api/v1/auth/session` → `200 { "mode": "protected", "user": { "username": "alice", "role": "user" } }`. `mode` is `open` or `protected`; `user` is `null` without a valid session; `role` is `admin` or `user`.

Since 2026-11-10 (issue #115, [technical spec](doc/specifications/app-information/technical.md)) the response also carries `version` (string, always present, never null): the release tag `vYYYYMMDDHHMMSS` embedded at build time, `<latest tag>-dev` for development builds, or `dev` when no tag is available. Example: `{ "mode": "open", "user": null, "version": "v20261109200945" }`. Rationale: additive, backward-compatible field on an endpoint every shell already loads in all modes, so no new endpoint is needed.

### Log in

`POST /api/v1/auth/login` with `{ "username": "alice", "password": "..." }`; username matched case-insensitively.

- `200 { "user": { "username": "alice", "role": "user" } }`, sets the cookie and records the last login time.
- `400 INVALID_JSON`, `401 INVALID_CREDENTIALS`, `429 LOGIN_LOCKED`, `429 LOGIN_BUSY`. No error sets a cookie.

Since 2026-11-23 (issue #266, [technical spec](doc/specifications/login-attempt-limits/technical.md)); request and success shapes unchanged:

- Client address: the `CF-Connecting-IP` request header when it is a bare valid IPv4 or IPv6 address, else the host of the connection address (port ignored). The header is trusted because the application is reachable only through Cloudflare.
- Failures (`401`) are counted per pair of client address and lowercased username; unknown usernames are counted too. A success clears its pair only. 5 consecutive failures lock the pair for 1 minute from the 5th failure: its attempts get `429 LOGIN_LOCKED` at once, without a password check, are not counted and do not extend the lock. Failures of an unlocked pair are forgotten after 1 hour without a new failure; ended locks are forgotten.
- Password checks run one at a time. When 4 logins already wait for one, a further login that needs a check gets `429 LOGIN_BUSY` at once. The lock is checked first, so a locked pair gets `LOGIN_LOCKED`, never `LOGIN_BUSY`. Busy and cancelled logins are not counted.
- `400 INVALID_JSON` is not counted.

### Log out

`POST /api/v1/auth/logout` → `204`; deletes the current session if any and clears the cookie.

### List users (administrator)

`GET /api/v1/admin/users` → `200 { "users": [ { "username": "alice", "lastLoginAt": "2026-10-04T10:00:00Z" } ] }`. Regular users only, ordered by username; `lastLoginAt` is `null` before the first login.

### Add a user (administrator)

`POST /api/v1/admin/users` with `{ "username": "alice", "password": "at-least-12-chars" }`.

| Status | Code | When |
| --- | --- | --- |
| 201 | — | `{ "user": { "username": "alice", "lastLoginAt": null } }`. The first user switches to protected mode and receives open-mode items and settings atomically. |
| 400 | `INVALID_JSON` | Invalid body or fields not strings |
| 400 | `INVALID_USERNAME` | Not 3–32 letters, digits, `.`, `-`, `_` |
| 400 | `PASSWORD_TOO_SHORT` | Fewer than 12 characters |
| 400 | `PASSWORD_TOO_LONG` | More than 256 characters |
| 409 | `USERNAME_TAKEN` | Same username ignoring case, including `admin` |

### Administrator account

`admin` is created, or its password reset, only on the device with `pricefollower admin-password`. No endpoint changes passwords in phase 1.

## LeBoncoin old price (2026-10-05)

Issue #42; see [technical specification](doc/specifications/leboncoin-old-price/technical.md). Additive change to item-price observation objects (`latestPrice`, `lastThreeDetections[]`, `priceHistory[]`); Amazon second-hand detections are unchanged. No new endpoint, status code, or error.

```json
{ "amountCents": 32000, "currency": "EUR", "timestamp": null, "oldPrice": true }
```

- `oldPrice` (boolean, always present): `true` for the listing's old price recorded once when a LeBoncoin item is added and its first collection succeeds; `false` for every collected observation.
- `timestamp` (string or `null`): `null` only when `oldPrice` is `true` and the listing gave no date; the UI then shows "Old price" in place of the date. A dated old price carries that date.
- Ordering: undated old prices are older than every other observation; dated ones are ordered by date. Old prices take part in `priceHistory` and in the `lastThreeDetections` periods but never change the latest collection attempt or `status`.
- Rationale: the user asked for a historical price rather than a new column; a flag on the existing observation keeps one price history and avoids a new item field.

## Amazon searches (2026-10-07) — removed (issue #111)

**Removed by issue #111 (2026-11-10); kept below only as history, not part of the current contract.** The routes `/api/v1/amazon-searches…` and `GET /api/v1/amazon/requests` now return `404 ROUTE_NOT_FOUND`; the item field `tracked`, `409 ITEM_NOT_TRACKED`, `409 AMAZON_REQUESTS_STOPPED`, Amazon `aiReview` objects (Amazon details return `aiReview: null` again), the search-item reuse on add and the untrack-on-delete behavior are gone. The current Amazon stop contract is "Amazon per-item stop (issue #111)" below. Rationale: [change index](doc/changes/2026-11-10-0141-issue-111-delete-amazon-searches/index.md).

Issue #56, agent-defined per AGENTS.md. Technical sources: [Amazon searches](doc/specifications/amazon-searches/technical.md), [Amazon human browsing](doc/specifications/amazon-human-browsing/technical.md), [Amazon AI review](doc/specifications/amazon-ai-review/technical.md). Rationale: [change index](doc/changes/2026-10-07-1117-issue-56-amazon-searches/index.md). Additive except where noted. All endpoints follow the existing access rules (owner-scoped in protected mode, open-mode owner otherwise); another owner's search returns `404 SEARCH_NOT_FOUND`. Responses carry `Cache-Control: no-store`.

### Item response changes

- `tracked` (boolean, always present): `true` for items of the Amazon and LeBoncoin tabs; `false` for Amazon items that exist only as search items. Existing items are `true`.
- `GET /api/v1/items` and `POST /api/v1/items/refresh` cover tracked items only (unchanged for existing data). `GET /api/v1/items/{id}` also returns search items.
- `POST /api/v1/items` with an Amazon URL whose product is already a search item of the user makes it tracked, keeping history and reviews, and returns `201` with that item; it is collected immediately as a new item.
- `DELETE /api/v1/items/{id}` on a tracked item that still belongs to a search of the user only untracks it (`204`); it stays in the search with its history. An untracked item returns `404 ITEM_NOT_FOUND`.
- `POST /api/v1/items/{id}/refresh`: `409 ITEM_NOT_TRACKED` ("Search items are checked during the Amazon request windows.") for an untracked item; `409 AMAZON_REQUESTS_STOPPED` ("Amazon requests are stopped after repeated failures. Restart them from the Amazon searches tab.") for an Amazon item while Amazon requests are stopped. `POST /api/v1/items/refresh` keeps its response; Amazon items are skipped without an attempt while stopped.
- `aiReview` in `GET /api/v1/items/{id}` is an object for Amazon items that belong to a search of the user or have reviews, and `null` for other Amazon items. The state and review object fields are unchanged; an Amazon `review` contains only `{ "price": { "rating": "good_deal | fair | overpriced", "explanation": "…" } }` (English). `priceCents` is the price the review was based on. No review starts when the price changes; clients compare `latestPrice.amountCents` with `latest.priceCents` to show "price changed". `POST /api/v1/items/{id}/ai-review` still returns `422 AI_REVIEW_UNSUPPORTED` for Amazon items.

### Amazon requests object

```json
"amazonRequests": { "stopped": false, "stoppedAt": "2026-10-07T21:40:00Z", "consecutiveFailures": 0 }
```

| Field | Type | Meaning |
| --- | --- | --- |
| `stopped` | boolean | `true`: no Amazon request is sent (searches and tracked Amazon items) until a search refresh restarts them. Application-wide. |
| `stoppedAt` | string or `null` | Time of the latest stop. Kept after a restart until the next successful Amazon request, then `null`. The UI shows the stop information line while non-null. |
| `consecutiveFailures` | integer 0–5 | Consecutive failed Amazon requests; reset by any successful request. |

`GET /api/v1/amazon/requests` → `200 { "amazonRequests": {…} }`; other methods `405` (`Allow: GET`).

### Search object

```json
{
  "id": "5f0c…",
  "url": "https://www.amazon.fr/joursprime/?_encoding=UTF8&…",
  "label": "amazon.fr/joursprime/",
  "addedAt": "2026-10-07T13:00:00Z",
  "capturedAt": null,
  "itemCount": 0,
  "state": "waiting",
  "waitingUntil": "2026-10-07T20:00:00Z",
  "lastError": null
}
```

| Field | Type | Meaning |
| --- | --- | --- |
| `url` | string | URL as entered (trimmed), never modified. |
| `label` | string | Host without `www.` followed by the path. |
| `capturedAt` | string or `null` | When the first 30 items were captured; `null` before. The item set never changes afterwards. |
| `itemCount` | integer 0–30 | Captured items. |
| `state` | `waiting` \| `running` \| `done` \| `stopped` | `waiting`: first retrieval or next pass due, waiting for a request window or its turn. `running`: being processed. `done`: the latest pass is finished. `stopped`: Amazon requests are stopped and this search has unfinished work (Refresh offered). |
| `waitingUntil` | string or `null` | Start of the next request window (22:00 or 06:00 Europe/Paris, as UTC) when `state` is `waiting` outside a window; else `null`. |
| `lastError` | object or `null` | `{ "at": "…", "message": "The Amazon search page could not be retrieved." }` for the latest failed results-page request; `null` after a success. Never contains upstream content. |

### List searches

`GET /api/v1/amazon-searches` → `200 { "searches": [ … ], "amazonRequests": {…} }`, most recently added first.

### Add a search

`POST /api/v1/amazon-searches` with `{ "url": "https://www.amazon.fr/joursprime/?…" }` (body ≤ 1 MiB). Stored immediately; this call makes no Amazon request. The first retrieval happens in a request window.

| Status | Code | When | Message |
| --- | --- | --- | --- |
| 201 | — | Search object, `state: "waiting"` | |
| 400 | `INVALID_JSON` | Invalid body, several values, `url` not a string | "Request body must be valid JSON." |
| 400 | `INVALID_URL` | Empty, longer than 2048 characters, or not an `http`/`https` URL | "Enter an Amazon search URL starting with https://." |
| 422 | `UNSUPPORTED_SEARCH` | Host is not `amazon.de`, `.fr`, `.es`, `.it`, `.nl`, `.be` (optionally `www.`) | "Only searches on the supported Amazon euro marketplaces can be added." |
| 409 | `SEARCH_ALREADY_ADDED` | Same trimmed URL already added by the user | "This search is already added: amazon.fr/joursprime/." |
| 413 | `REQUEST_TOO_LARGE` | Body over 1 MiB | "Request body is too large." |
| 405 | `METHOD_NOT_ALLOWED` | Other methods on the collection (`Allow: GET, POST`) | "This method is not allowed for the route." |

### Get a search and its items

`GET /api/v1/amazon-searches/{id}` → `200 { "search": {…}, "amazonRequests": {…}, "items": [ … ] }`; `404 SEARCH_NOT_FOUND` ("Amazon search was not found.").

`items` are in captured (Amazon) order. Each is an item response (list shape, without `priceHistory`) plus:

```json
"position": 1,
"aiReviewSummary": { "status": "available", "priceRating": "good_deal", "priceCents": 10000 }
```

`aiReviewSummary.status`: `none`, `pending`, `available`, `failed`, `no_token`. `priceRating` (`good_deal`, `fair`, `overpriced`) and `priceCents` come from the latest succeeded review, else `null`. A product no longer available has item `status: "unavailable"` and keeps its last price.

### Delete a search

`DELETE /api/v1/amazon-searches/{id}` → `204`; `404 SEARCH_NOT_FOUND`. Deletes the search and its items with their price history and reviews, except items that are tracked or belong to another search of the user. No further request is made for it.

### Move an item to the tracked Amazon items

`POST /api/v1/amazon-searches/{id}/items/{itemId}/track` (no body) → `200` item response with `tracked: true` and `nextCheckAt` set. It keeps its history and reviews and stays in the search. `404 SEARCH_NOT_FOUND`; `404 ITEM_NOT_FOUND` when the item is not in that search; `409 ITEM_ALREADY_TRACKED` ("This listing is already being tracked.").

### Refresh a search after a stop

`POST /api/v1/amazon-searches/{id}/refresh` (no body) → `202 { "requestedAt": "…", "search": {…} }`. Restarts all Amazon requests (failure count reset) and starts a new pass of this search, AI reviews included for items without a succeeded review; history and reviews are kept. Outside a window the search is `waiting` until the next one. `404 SEARCH_NOT_FOUND`; `409 AMAZON_NOT_STOPPED` ("Amazon requests are not stopped.") when `stopped` is false and `stoppedAt` is `null`.

Other methods on the routes above return `405 METHOD_NOT_ALLOWED` with the matching `Allow` header.

### Timing (informative, not configurable)

Search work runs only 22:00–01:00 and 06:00–08:00 Europe/Paris, one search at a time and one item at a time, with random 30–120 s pauses; each search gets one pass per window. A failed request is retried once at the end of the pass. Five consecutive failed Amazon requests from any source set `stopped`.

## Amazon per-item stop (issue #111)

Agent-defined per AGENTS.md. Technical sources: [Amazon request stop](doc/specifications/amazon-request-stop/technical.md), [Amazon searches removal](doc/specifications/amazon-searches-removal/technical.md). Rationale: [change index](doc/changes/2026-11-10-0141-issue-111-delete-amazon-searches/index.md).

### Item response change

- `amazonStoppedAt` (string or `null`, always present in list, add and details responses): when the Amazon item was stopped after 5 consecutive failed Amazon requests; `null` when not stopped and for LeBoncoin items. The stop is application-wide in effect (the 5th failure stops the Amazon items of all owners), the field is owner-scoped like the item.
- The Amazon tab shows a warning while any listed Amazon item has a non-null `amazonStoppedAt`.

### Effect on existing endpoints

- `POST /api/v1/items/{id}/refresh` on a stopped Amazon item → `202` as usual; the check is sent despite the stop. An Amazon answer (success, price not found, unavailable) sets only that item's `amazonStoppedAt` to `null`; a failure leaves it set. Clients poll `GET /api/v1/items` as for any refresh.
- `POST /api/v1/items/refresh`: response unchanged (`itemsQueued` counts every tracked item); stopped Amazon items are skipped without a request or a new attempt. Scheduled checks skip them too.
- `POST /api/v1/items` adds a new Amazon item with `amazonStoppedAt: null`, even while other items are stopped.

## Amazon human-like checks (issue #168)

Agent-defined per AGENTS.md. Technical source: [Amazon human-like checks](doc/specifications/amazon-human-checks/technical.md). No new endpoint, field or error code; request and response shapes are unchanged. Only the meaning of existing values for Amazon items changes:

- `nextCheckAt` of an Amazon item is a random time (second resolution) inside the next Europe/Paris window, 06:00–09:00 or 19:00–23:00 (the configured timezone's windows if another is set), drawn separately for each item and anew after each scheduled check. `PRICEFOLLOWER_CHECK_TIMES` applies to LeBoncoin items only. `nextCheckAt` may be `null` for a moment at start-up after the upgrade, as already allowed.
- `POST /api/v1/items` (Amazon): the first check starts immediately as before; `nextCheckAt` is in the first window that starts after the add.
- `POST /api/v1/items/{id}/refresh` and `POST /api/v1/items/refresh`: responses unchanged; they no longer change a future `nextCheckAt` of Amazon items (LeBoncoin unchanged). `POST /api/v1/items/{id}/refresh` (Refresh price) moves an already passed Amazon `nextCheckAt` (a check skipped while the item was stopped) to a random time in the next window; `POST /api/v1/items/refresh` never changes Amazon `nextCheckAt`.
- Upgrade: Amazon items stopped before this version are restarted once (`amazonStoppedAt: null`) with a `nextCheckAt` in the next window; stops occurring later follow "Amazon per-item stop (issue #111)".
- A failed browser check of an Amazon item is reported as before (`lastAttempt.result: "request_error"`, message "Amazon could not be reached for a price check.").
- Issue #186: an Amazon answer with a success status (200–299) that the browser does not open as a page (a download such as `Content-Disposition: attachment`, or 204/205 without content) is such a failure: `lastAttempt.result: "request_error"`, item `status: "retrieval_error"`, second-hand status `check_error`, latest price and second-hand offer unchanged, and it counts toward the per-item stop. It is never `price_not_found`. A non-2xx answer that is not opened as a page is still classified by its status (404/410 → unavailable, other statuses → `request_error`). Refresh endpoints still answer `202`.
- Issue #187: when such a non-opened answer is 404/410 (listing unavailable: `lastAttempt.result: "unavailable"`, item `status: "unavailable"`), no used-offer request is sent for that check; its second-hand status is `check_error` and the last detected second-hand offer is unchanged. A 404/410 page that the browser opens still gets its used-offer requests. Refresh endpoints still answer `202`.

## HTTP server limits (issue #273)

Agent-defined per AGENTS.md. Technical source: [HTTP server timeouts](doc/specifications/http-server-timeouts/technical.md). No new endpoint, field or JSON error code; every request and response shape is unchanged. The limits apply to every request (API, pages, assets), in development and production mode, before or around the endpoint rules above:

| Limit | Value | When it is passed |
| --- | --- | --- |
| Request headers received | 5 s from the request start (unchanged) | The connection is closed with no response or, when part of the request line has arrived, after Go's plain-text `400 Bad Request`. The request is not handled. |
| Whole request (headers and body) received | 15 s from the request start | For an endpoint that reads a request body (`POST /api/v1/auth/login`, `POST /api/v1/items`, `PUT /api/v1/items/{id}/purchase-goal`, `PUT /api/v1/settings`): the connection is closed; no response, and the request is not handled (nothing saved or started). Any other request is handled once its headers arrive, as before; if a body it announces but does not read is not received within 15 s, its response is sent with `Connection: close`. |
| Response sent | 60 s from the received request headers (includes reading the body and handling) | The connection is closed; the client gets no response or a cut-off one. What the request did is kept. |
| Idle kept-alive connection | 60 s (unchanged) | The connection is closed. |
| Request headers size | 64 KiB accepted; over 68 KiB (69,632 bytes) refused | `431 Request Header Fields Too Large`, then the connection is closed; the request is not handled. |

- The request start is when the connection is accepted, or, on a kept-alive connection, when the first bytes of the next request arrive.
- The `431` answer is written by Go's `net/http` before any routing, so it does not use the JSON error shape: `Content-Type: text/plain; charset=utf-8`, `Connection: close`, body `431 Request Header Fields Too Large`. It is returned with or without a session, for API and page paths alike.
- The `400 Bad Request` for a request line cut by the 5 s header limit is the other answer that `net/http` writes before routing. It is also plain text and not in the JSON error shape (Go's behaviour before this change too).
- Endpoint body limits are unchanged (`413 REQUEST_TOO_LARGE`: 4 KiB login, 32 KiB settings, 1 MiB items and purchase goal).
- No endpoint waits for marketplace checks or AI reviews; `PUT /api/v1/settings` waits at most for the 20 s Claude verification, well within 60 s.
- The frontend has no request time limit of its own; a connection closed by these limits shows its existing network error state.

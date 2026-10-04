# Functional Specifications — Price Tracking Service

## 1. Purpose

The service tracks the price of an item listed on a European Amazon marketplace or LeBoncoin France. The operator adds an item by providing its listing URL. The server checks its price twice per day, stores observations, and serves the information to a web interface.

This document defines the first version (v1). Other marketplaces and possible enhancements are listed under [Next steps](#12-next-steps). Remaining decisions are listed under [Open decisions](#13-open-decisions).

Related documents: [API contract](../API_SPECIFICATION.md) · [Use case index](use-cases/README.md).

## 2. V1 goals and scope

- Add a supported European Amazon or LeBoncoin listing by URL.
- Check each active item twice per day and store detected prices with timestamps.
- Show all tracked items and their latest known prices in one compact Prices column, with up to three recent price periods that combine consecutive checks at the same price.
- Show the latest second-hand offer sold by Amazon, when available, with its condition and recent detections.
- Allow the operator to manually refresh the prices for all tracked items.
- Allow the operator to manually refresh an individual item from its details page.
- Allow the operator to inspect the complete price history for a tracked item and delete an item.
- Keep collection attempts running when a price is stale or a check fails.

V1 supports euro-priced listings from Amazon Germany, France, Spain, Italy, the Netherlands, and Belgium, plus LeBoncoin France. The expected maximum is approximately 100 tracked items. There is one shared server-side collection of items; no login or per-user accounts are required. Optional local users and login are specified in [multiple local users and login](specifications/multi-user-login/functional.md); while no regular user is created, the application keeps working as described here. The server owns item storage and price collection. The front end displays data supplied by the server and submits add, delete, and refresh actions.

Prices are the item price only, in euros. Shipping, taxes, and other charges are excluded. V1 assumes one price per listing; price ranges are not represented. Notifications, price trends, and login are out of scope.

## 3. Core concepts

**Tracked item:** A saved reference to an eligible Amazon or LeBoncoin listing, including its platform, URL, display information, tracking status, and observations.

**Price observation:** A successful detection containing the item price in euros and the detection timestamp. Collection failures are recorded separately and do not overwrite the last successful price.

**Latest price:** The value and timestamp from the latest successful observation. It may be stale if later checks have failed or been delayed.

**Price period:** A run of consecutive successful observations with the same price. The main list shows each recent period once, using the timestamp of that price's latest successful observation. A new amount starts a new period, including when the amount returns to an earlier value after changing.

## 4. Technical architecture constraints

- The application has a separate front end and backend.
- The backend must run on Go and is responsible for the service API, scheduled price collection, and persistence.
- Use SQLite for server-side storage of tracked items, price observations, and collection outcomes as needed.
- The front end must use React 19 and IBM Carbon Design System components and visual conventions.
- The front end reads and displays data from the backend; it does not connect directly to SQLite or collect marketplace prices.
- Keep collection logic in separate platform adapters for Amazon and LeBoncoin.
- Use an API-first workflow for cross-tier features: define the API contract and obtain operator confirmation before implementing either backend or frontend changes. After confirmation, implement both tiers against the approved contract.
- Provide a development mode that starts the backend and serves the frontend together, so a developer can open the application and use its pages and backend API without manually starting each tier.
- Development mode must provide repeatable sample data in a development SQLite database so the interface can be reviewed without relying on live marketplace requests. Keep development seed data isolated from any non-development database.
- Development mode must allow the developer to navigate directly to the required frontend page (including a detail page where applicable) and have that page request and display data from the backend.

The API shape, Go HTTP implementation, database access library, scheduling implementation, and deployment arrangement are not prescribed here and can be selected during technical design. An API contract for a cross-tier change should cover endpoints and methods, request/response data, validation/errors, and any client-visible states.

## 5. Main user journeys

### 5.1 Add an item

1. The operator opens the tracked-items page and chooses **Add item**.
2. The operator pastes a supported Amazon European or LeBoncoin France listing URL and submits it.
3. The server validates the URL and confirms that it is an eligible listing.
4. The server attempts to identify the item and immediately fetch its price; the operator should not have to wait for the next scheduled check for the first attempt.
5. The item appears in the list if accepted. Its initial price may be pending if the immediate retrieval is still in progress or did not succeed.
6. The interface explains invalid, unsupported, duplicate, or unprocessable URLs.

### 5.2 Review tracked items

The main page shows all tracked items. Each row/card includes:

- Item title, or a fallback label if unavailable.
- Source platform and listing link.
- Thumbnail when available.
- One compact Prices column containing up to three recent consecutive price periods, newest first. The latest price appears once as the first period; each amount is paired with the time of its latest successful observation. Repeated checks at the same price update that period's time; a changed price starts a new period.
- An explicit unavailable or awaiting-first-price state when no successful price is available.
- Collection/listing status, such as active, stale, retrieval issue, or unavailable.
- Latest second-hand offer sold by Amazon for Amazon items, including price and condition; third-party offers are ignored. This field is not applicable to LeBoncoin items.
- Actions to open item details, delete an item, or refresh all tracked items.

The page should have clear empty, loading, and error states. A stale price does not stop scheduled checks or retries. The stale threshold is an open decision.

### 5.3 View item details

The detail view shows item metadata, platform, listing URL, latest known price and timestamp, current status, and every successful price detection for as long as the item has been tracked, newest first. Amazon items also show every successful Amazon-sold second-hand offer detection with its condition. The operator can scroll through the full lists and request an immediate price refresh from this page; the interface displays progress and then reloads the server result. V1 does not require a chart, trend calculation, or arbitrary date-range selection.

### 5.4 Delete an item

The operator can delete an item from tracking. Deletion removes the item and its associated price observations from the server database. No history of deleted items is retained. V1 does not require pause or archive actions.

## 6. Price collection behavior

- Each active item is scheduled for two collection attempts per day.
- As soon as a new item is accepted, trigger an immediate price collection attempt; then continue with its regular twice-daily schedule.
- Preserve the submitted Amazon listing URL and use it for collection requests; normalize the Amazon product identity separately for duplicate detection.
- The schedule defaults to 08:00 and 20:00 in `Europe/Paris`; timezone and check times are configurable for deployment.
- Each successful check stores the detected item price in euros and its timestamp.
- Amazon checks also look for second-hand offers sold by Amazon. If multiple qualifying offers exist, store the lowest-priced offer and its condition. Ignore third-party offers; do not accept an offer unless Amazon's seller attribution can be verified. LeBoncoin items have no separate second-hand-offer field.
- A LeBoncoin free/donation listing with no numeric price is recorded as a zero-euro price only when the listing explicitly indicates that it is a donation/free item. The interface displays this as “Gratuit”. A missing or unparseable price is not treated as free.
- Record the second-hand check outcome as available, no qualifying offer found, pending, or check error. Preserve the last detected Amazon offer when a later check finds none or cannot verify the result.
- Each failed check records a useful outcome, such as temporary retrieval error, listing unavailable, or price not found.
- A failed check never replaces the latest successful price with a blank, zero, or inferred value.
- Continue scheduled retries when an item is stale or previous collection attempts failed.
- A manual refresh from an item's details page queues an immediate check for that item only. An in-progress check is not duplicated.
- Collection requests should present as access from a Google Chrome browser, as specified by the product owner. A challenge or other non-success response is a retrieval error, not a free price or proof that the listing is unavailable.
- Store every successful price check, including unchanged prices, and retain every observation while the item remains tracked.
- The server should expose when the next check is expected if useful to the interface.

## 7. Pages and interface areas

### Tracked items page

The primary page contains the add-item action, a search field, and the tracked-items list. Search filters locally by title, listing ID, ASIN, platform, or marketplace. Each row presents the latest price and recent price periods together in the Prices column. No sign-in is required.

### Item details page

Shows item metadata, recent price detections, collection status, source listing, refresh action, and deletion action.

### Add-item flow

A URL input with validation feedback, submission progress, and a clear success or error result. Its specific layout is not prescribed.

### Server-backed data

The server is the source of truth for tracked items and price observations. The front end retrieves and displays server data and submits add/delete/refresh actions. No user identity or ownership model is required in v1.

## 8. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-01 | The operator can submit an eligible European Amazon or French LeBoncoin listing URL to add an item. | Must |
| FR-02 | The server validates URL format and eligibility for v1. | Must |
| FR-03 | The server stores accepted items and associates price observations with them. | Must |
| FR-04 | Active items receive two scheduled price collection attempts per day. | Must |
| FR-05 | The list shows the latest successful item price and up to three recent consecutive price periods together in one Prices column, newest first, with each period's latest observation time. Repeated checks at the same price update that period's time; a different price starts a new period. | Must |
| FR-06 | The detail view shows all successful price observations for the full tracking period. | Must |
| FR-07 | Collection failures are recorded without corrupting the last successful price, and retries continue. | Must |
| FR-08 | The operator can open the source listing. | Should |
| FR-09 | The operator can delete an item and its associated price history. | Must |
| FR-10 | The front end displays server-provided data; storage and collection are server-side. | Must |
| FR-11 | The service supports approximately 100 tracked items in v1. | Should |
| FR-12 | Development mode starts the backend and serves the frontend together, and includes repeatable sample data in an isolated development database. | Must |
| FR-13 | Development mode supports direct navigation to frontend pages and successful data requests from those pages to the backend. | Must |
| FR-14 | Adding an accepted item triggers an immediate price collection attempt, followed by the regular twice-daily checks. | Must |
| FR-15 | Amazon item collections check for second-hand offers and only store offers explicitly sold by Amazon; third-party offers are excluded. This does not apply to LeBoncoin items. | Must |
| FR-16 | The main list shows the latest Amazon-sold second-hand offer price and condition, or a clear pending/not-found/error state. | Must |
| FR-17 | The operator can request a check of all tracked items from the main page; the API accepts asynchronously and the interface shows progress. | Must |
| FR-18 | The operator can request an immediate check of one item from its details page; the API accepts asynchronously and the interface shows progress. | Must |
| FR-19 | An explicitly donated/free LeBoncoin listing with no numeric price is recorded as zero euros and displayed as “Gratuit”; missing prices are not assumed to be free. | Must |
| FR-20 | The item details view shows every successful item-price observation for the full tracking period; Amazon details also show every successful Amazon-sold second-hand offer observation. | Must |

## 9. Important states and edge cases

- URL is malformed or does not belong to a supported Amazon or LeBoncoin listing.
- URL is valid but the listing cannot be retrieved or parsed.
- Item is added but the initial price is pending.
- The immediate collection attempt after adding an item fails; the item remains tracked and scheduled retries continue.
- The listing is already being tracked.
- The source listing is removed, sold, expired, or made private.
- A price is missing, unchanged, promotional, or ambiguous.
- Listing content changes, redirects, or points to a different item.
- A scheduled check is delayed or fails.
- The latest successful price is stale, but retries continue.
- Amazon blocks or limits a collection request.
- LeBoncoin blocks a request with a JavaScript challenge or returns a non-success response; record a retrieval error and continue retries.
- A LeBoncoin listing has no numeric price but explicitly indicates a donation/free item.
- A LeBoncoin listing has no price and no explicit donation/free text; report price not found, not zero.

The interface should preserve and clearly label the latest successful observation when later checks fail.

## 10. Non-goals for v1

- User accounts or login.
- Marketplaces other than European Amazon and LeBoncoin France (Temu, Vinted, and Wallapop remain future work).
- Price trends, percentage changes, or charts.
- Notifications or target-price alerts.
- Shipping/tax-inclusive prices, currency conversion, or price ranges.
- Retaining records after an item is deleted.
- Export of data or per-user privacy controls.

## 11. Data retention and capacity

Keep price observations for as long as the item remains tracked, including several months of observations. Deleting an item deletes its history. The initial expected tracked-item capacity is around 100; a larger limit may be introduced in a later version.

## 12. Next steps

Potential later marketplace support:

- Temu
- Vinted
- Wallapop

Each platform will need its own supported-region rules, collection behavior, and handling of listing-specific states. Potential later product enhancements include trends, charts, notifications, additional currencies, and a higher item limit.

## 13. Open decisions

1. **Amazon coverage:** Are the currently supported European Amazon country domains and listing forms sufficient?
2. **Schedule:** The implementation defaults to 08:00 and 20:00 in `Europe/Paris`; should the production deployment use different configurable check times or a different timezone?
3. **Price parsing:** How should promotional, coupon, or multi-option prices be interpreted while only storing the item price?
4. **Staleness:** After how long without a successful check should a price be marked stale?
5. **Item limit:** Should the approximately 100-item limit be enforced, and what should happen when it is reached?
6. **Failures:** How should repeated failures and unavailable listings be surfaced? Should retries use a defined backoff?
7. **Presentation:** Which language, locale formatting, and accessibility target should the interface use?

## 14. Initial acceptance criteria

- The operator can submit an eligible European Amazon or French LeBoncoin URL and receive a clear result.
- An accepted item appears in the list, including when its initial price is pending.
- A successful check creates a timestamped euro price observation.
- A qualifying Amazon-sold second-hand offer is recorded with its timestamp and condition; third-party offers are never shown as Amazon offers.
- The main list distinguishes a current Amazon-sold offer, no qualifying offer, a pending check, and an uncertain/failed check.
- The operator can manually refresh all tracked items and see pending progress until the checks finish.
- The operator can refresh an individual item from its details page and see pending progress until the check finishes.
- The operator can track an eligible LeBoncoin France ad, including the example priced item and the explicit free/donation item.
- A free/donation observation is stored as zero euros and shown as “Gratuit”; an absent or unclear price is not shown as free.
- Adding an accepted item immediately triggers its first collection attempt without waiting for the twice-daily schedule.
- Active items have two collection attempts per day, and retries continue when checks fail or prices become stale.
- The list shows the latest successful price once, together with up to three recent consecutive price periods in one Prices column, or a clear unavailable state.
- The item details view displays all successful item-price detections, and all successful Amazon-sold second-hand detections when applicable.
- Failed checks do not erase or misrepresent the last successful price.
- Deleting an item removes its stored price history.
- No login is required; the server stores data and collects prices, while the front end displays server-provided information.
- The backend runs on Go and persists data in SQLite.
- The separate front end uses React 19 and IBM Carbon Design System.
- A developer can start one development mode, open the frontend directly at the tracked-items or item-details page, and see sample data returned by the backend.
- Development seed data is repeatable and cannot overwrite the non-development database.

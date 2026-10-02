# UC-PC-01 — Collect and record prices

**Primary actor:** PriceFollower scheduler or an immediate collection request
**Supporting systems:** Amazon or LeBoncoin collector, SQLite store
**Trigger:** A scheduled check is due, a listing is added, or a manual refresh is accepted.

## Preconditions

- The item is stored in the tracked collection.
- Its platform collector is configured for the supported listing.

## Main flow

1. The service requests the listing price through the matching platform collector.
2. For Amazon items, the collector also checks for offers explicitly sold by Amazon and selects the lowest-priced qualifying offer when available. Third-party offers are excluded.
3. For LeBoncoin, an explicit free/donation listing with no numeric price is recorded as zero euros; an absent or unparseable price is reported as not found.
4. On a successful item-price check, the store retains a timestamped observation in integer euro cents.
5. The API's latest price reflects the newest successful observation. The list summarizes consecutive observations at the same amount as one price period, using that period's newest observation time.
6. A different amount begins a new period. Returning to an amount seen before a different amount also begins a new period.
7. Amazon offer observations are retained separately with condition and time. Offer status distinguishes an available offer, no qualifying offer, a pending check, and an uncertain or failed check.

## Alternatives and errors

- A request error, unavailable listing, or price-not-found result records the attempt outcome but does not create a successful price observation.
- A failed check does not erase or replace the latest successful item price or latest qualifying Amazon offer.
- Checks continue on the twice-daily schedule after stale data or collection failures.
- Repeated successful observations at an unchanged price remain in full item history even though the list groups that consecutive price period into one row.

## Postconditions

- The item retains all successful observations while it remains tracked, along with collection attempt outcomes.
- The tracked-items list can show current price periods without duplicating consecutive unchanged prices.

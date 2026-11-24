# UC-AMZ-THUMB-003 — Re-check an Amazon item that already has a thumbnail

**Primary actor:** PriceFollower application (scheduled check), or the user who chooses **Refresh price** or **Refresh all prices**
**Supporting systems:** Browser, server's Amazon check, Amazon product page, SQLite store
**Trigger:** A successful Amazon check of a tracked item that already has a stored thumbnail (UC-AMZ-HCHK-001, UC-AMZ-HCHK-003, UC-ID-02, UC-TI-03).
**Origin:** Issue #275.
**Covers:** FR-AMZ-THUMB-001, FR-AMZ-THUMB-002, FR-AMZ-THUMB-003, FR-AMZ-THUMB-004, FR-AMZ-THUMB-005

## Preconditions

- An Amazon item is tracked with a stored thumbnail, either an accepted Amazon image URL (for example `https://m.media-amazon.com/images/I/old.jpg`) or a URL stored before this change that is not accepted (for example `http://example.com/old.jpg`).

## Main flow

1. The check opens the item's product page and finds the price.
2. The page offers an accepted Amazon image URL, for example main image `https://m.media-amazon.com/images/I/new.jpg`.
3. The check records that URL as the thumbnail, replacing the stored one, including a stored URL that is not accepted (FR-AMZ-THUMB-001, FR-AMZ-THUMB-005).
4. The tracked-items list and the item details show `…/new.jpg`.

## Alternatives and errors

- **Earlier candidate rejected:** a rejected main image is skipped and the first accepted later candidate replaces the stored thumbnail (FR-AMZ-THUMB-002).
- **No candidate accepted:** the check records no thumbnail; the item keeps its stored thumbnail, for example `…/old.jpg` (FR-AMZ-THUMB-003). A stored URL that is not accepted is also kept until a later check finds an accepted one (FR-AMZ-THUMB-005).
- **Rejected image in any case:** the price, title, second-hand offer, status and next check are recorded as usual, with no error or warning for the image (FR-AMZ-THUMB-004).
- **Check fails:** the thumbnail is unchanged and the failure behaves as in UC-AMZ-HCHK-005.

## Postconditions

- The item's thumbnail is the first accepted Amazon image URL of the latest page, or the previously stored thumbnail when the page offered none.
- Thumbnails stored before this change are not removed or rewritten except by a later check that finds an accepted Amazon image URL.

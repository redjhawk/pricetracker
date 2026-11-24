# UC-AMZ-THUMB-002 — Add an Amazon item whose page offers no accepted Amazon image

**Primary actor:** User who tracks Amazon items
**Supporting systems:** Browser, PriceFollower application, server's Amazon check, Amazon product page
**Trigger:** The actor adds an Amazon listing and the first check succeeds, but no page image candidate is an accepted Amazon image URL.
**Origin:** Issue #275: an image URL pointing anywhere was stored and then loaded by the tracked-items list in the actor's browser.
**Covers:** FR-AMZ-THUMB-003, FR-AMZ-THUMB-004

## Preconditions

- The actor is logged in (protected mode) or uses the application in open mode.
- The listing's product page shows a valid price, and every image it offers is rejected, for example only `https://example.com/a.jpg`, `http://m.media-amazon.com/…` or a `data:` URL.

## Main flow

1. The actor adds the Amazon listing.
2. The server's Amazon check opens the product page and finds the price and title.
3. The check examines the page image candidates in order and rejects each one.
4. The check records no thumbnail, exactly as for a page without any image (FR-AMZ-THUMB-003); the item's `thumbnailUrl` is empty (`null`).
5. The check records the price, title, second-hand offer, status and next check as usual; the item status is not **Retrieval error** and the latest attempt has no error message (FR-AMZ-THUMB-004).
6. The tracked-items list shows the item with the existing thumbnail placeholder and makes no request to the rejected image's host.

## Alternatives and errors

- **Page without any image:** same result as steps 4 to 6.
- **One candidate accepted:** see UC-AMZ-THUMB-001.

## Postconditions

- The item has no thumbnail and the list shows the placeholder.
- The check otherwise succeeded; no error or warning about the image is shown, and no screen, message or setting changed.

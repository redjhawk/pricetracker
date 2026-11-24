# UC-AMZ-THUMB-001 — Add an Amazon item whose page offers an Amazon image over HTTPS

**Primary actor:** User who tracks Amazon items
**Supporting systems:** Browser, PriceFollower application, server's Amazon check, Amazon product page
**Trigger:** The actor adds an Amazon listing (UC-IM-01, UC-AMZ-HCHK-002) and the first check of the item succeeds.
**Origin:** Issue #275: the Amazon check recorded the page's image URL as thumbnail without checking its scheme or host.
**Covers:** FR-AMZ-THUMB-001, FR-AMZ-THUMB-002, FR-AMZ-THUMB-004

## Preconditions

- The actor is logged in (protected mode) or uses the application in open mode.
- The listing is on a supported Amazon European marketplace and its product page offers at least one accepted Amazon image URL (starting exactly with `https://m.media-amazon.com/` or `https://images-eu.ssl-images-amazon.com/`).

## Main flow

1. The actor adds the Amazon listing.
2. The server's Amazon check opens the product page and finds the price and title.
3. The check takes the page image candidates in order: main product image, first product image with dynamic sizes, `og:image`.
4. The first candidate is an accepted Amazon image URL, for example `https://m.media-amazon.com/images/I/price-found.jpg`; the check records it as the item's thumbnail, with its path and query string as found (FR-AMZ-THUMB-001).
5. The tracked-items list shows the item with that image.

## Alternatives and errors

- **Main image on the second Amazon host:** the main image is `https://images-eu.ssl-images-amazon.com/images/I/x.jpg`; that URL is the thumbnail (FR-AMZ-THUMB-001).
- **Earlier candidate rejected:** the main image is `http://m.media-amazon.com/images/I/a.jpg` (or any other rejected URL: another host, `https://m.media-amazon.com.example.com/…`, other letter case, a port, user information, a `data:` URL, a relative path, an empty value) and a later candidate such as `og:image` is `https://m.media-amazon.com/images/I/b.jpg`; the rejected URL is never recorded and the thumbnail is `…/b.jpg` (FR-AMZ-THUMB-002).
- **No candidate accepted:** see UC-AMZ-THUMB-002.
- **Check fails (retrieval error, unavailable listing):** the image rule does not apply; the item behaves as in the Amazon check use cases (UC-AMZ-HCHK-005, UC-AMZ-BRET-001).

## Postconditions

- The item's thumbnail is the first accepted Amazon image URL of the page; no rejected URL is stored.
- Price, title, second-hand offer, status and next check are recorded as for any successful check (FR-AMZ-THUMB-004); no screen, message or setting changed.

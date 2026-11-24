# Functional specification: Amazon thumbnails only from Amazon image hosts over HTTPS

Status: ready (issue #275)
Owner: functional specification agent
User decision/reference: GitHub issue #275 ("Amazon: store only HTTPS thumbnail URLs from Amazon image hosts").

## Purpose and scope

An Amazon check records the image URL found in the product page as the item's thumbnail without checking where it points, and the tracked-items list then loads that URL in the operator's browser. The user asks that an Amazon check keep a thumbnail URL only when it is an HTTPS URL on an Amazon image host, as the LeBoncoin check already does for `https://img.leboncoin.fr/`. Otherwise the check records no thumbnail.

Actors: the operator viewing tracked items; the server's Amazon check (scheduled, at item addition, or on **Refresh price**).

Included: which image URL a successful Amazon check records as the item's thumbnail.

Excluded: the LeBoncoin thumbnail rule (unchanged); price, title, second-hand offer and status recording; the thumbnail placeholder and layout ([tracked-items-identity](../tracked-items-identity/functional.md)); thumbnails already stored before this change, which are not removed or rewritten by it (FR-AMZ-THUMB-005); development seed data; any new screen, message or setting.

Terms:

- **Accepted Amazon image URL:** a URL that begins exactly with `https://m.media-amazon.com/` or `https://images-eu.ssl-images-amazon.com/`. `m.media-amazon.com` is the host of the product images in the Amazon test pages (`internal/amazon/testdata/`); `images-eu.ssl-images-amazon.com` is named in the issue. Any other URL is rejected, for example `http://m.media-amazon.com/…`, `https://example.com/…`, `https://m.media-amazon.com.example.com/…`, a `data:` URL, a relative path or an empty value.
- **Page image candidates:** the image URLs the product page offers, in today's order: the main product image, then the first product image carrying a set of dynamic image sizes, then the page's `og:image` meta value.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-AMZ-THUMB-001 | A successful Amazon check of a product page whose first page image candidate is an accepted Amazon image URL. | The check records that URL as the item's thumbnail, as today. | With the `price-found.html` test page, the item's thumbnail is `https://m.media-amazon.com/images/I/price-found.jpg`, and the tracked-items list shows that image. A page whose main image is `https://images-eu.ssl-images-amazon.com/images/I/x.jpg` gives that thumbnail. |
| FR-AMZ-THUMB-002 | A successful Amazon check where a page image candidate is not an accepted Amazon image URL. | That candidate is never recorded as the thumbnail; the check uses the next candidate in order instead, and records the first accepted one. | A page whose main image is `http://m.media-amazon.com/images/I/a.jpg` and whose `og:image` is `https://m.media-amazon.com/images/I/b.jpg` gives the thumbnail `https://m.media-amazon.com/images/I/b.jpg`. |
| FR-AMZ-THUMB-003 | A successful Amazon check where no page image candidate is an accepted Amazon image URL. | The check records no thumbnail, exactly as for a page without any image: an item added by this check has no thumbnail and the list shows the existing placeholder; an item that already had a thumbnail keeps it. | Adding an Amazon item whose page offers only `https://example.com/a.jpg` (or `http://…`, or a `data:` URL) shows the placeholder and the item's `thumbnailUrl` is empty (`null`); no request to `example.com` is made by the list. Re-checking an item that has the thumbnail `https://m.media-amazon.com/images/I/old.jpg` with such a page leaves its thumbnail `…/old.jpg`. |
| FR-AMZ-THUMB-004 | An Amazon check whose image is rejected (FR-AMZ-THUMB-002, 003). | The rest of the check is unaffected: price, title, second-hand offer, status and next check are recorded as when the page has an accepted image, and no error or warning is shown for the image. | Checking a page with a valid price and only a rejected image records the price, the item status is not **Retrieval error**, and the latest attempt has no error message. |
| FR-AMZ-THUMB-005 | An Amazon item whose stored thumbnail is not an accepted Amazon image URL (stored before this change). | The thumbnail is kept until a later successful check finds an accepted Amazon image URL, which replaces it (FR-AMZ-THUMB-001, 002). | After a check of a page whose main image is `https://m.media-amazon.com/images/I/new.jpg`, an item that had `http://example.com/old.jpg` shows `…/new.jpg`. |

## States and corner cases

- The comparison is exact on the start of the URL: letter case, a port, user information or any other host than the two listed make the URL rejected.
- Query strings and paths after an accepted host are kept as found in the page.
- The rule applies to every Amazon check (at addition, scheduled, **Refresh price**) and to every supported Amazon European marketplace.
- No new loading, empty or error state: a rejected image looks like a listing without image (existing placeholder).

## Acceptance criteria summary

- An Amazon check records as thumbnail only a URL starting with `https://m.media-amazon.com/` or `https://images-eu.ssl-images-amazon.com/`, taking the first such candidate in the page's existing image order.
- When the page offers none, the check records no thumbnail and otherwise succeeds as usual.
- LeBoncoin thumbnails and all other Amazon check results are unchanged.

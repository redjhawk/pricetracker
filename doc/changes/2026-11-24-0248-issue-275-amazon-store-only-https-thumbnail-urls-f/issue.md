# Issue #275 Amazon: store only HTTPS thumbnail URLs from Amazon image hosts

**Problem:** `productImage` (`internal/amazon/collector.go`) stores the image URL found in the product page as the item thumbnail without checking its scheme or host, and the frontend loads it. The LeBoncoin collector already accepts only `https://img.leboncoin.fr/` URLs (`firstImage` in `internal/leboncoin/collector.go`).

**Suggested work:** Keep a thumbnail URL only when it is an `https` URL on an Amazon image host (for example `m.media-amazon.com` and `images-eu.ssl-images-amazon.com`; confirm the hosts against the fixtures in `internal/amazon/testdata/`). Otherwise the item has no thumbnail.


## Comments by trusted authors


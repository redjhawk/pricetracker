# UC-AMZ-BRET-001 — Amazon check of an available listing returns its price

**Primary actor:** PriceFollower application, on behalf of the user who tracks Amazon items
**Supporting systems:** Real web browser, Amazon, SQLite store
**Trigger:** An Amazon check starts (first check after add, scheduled check, **Refresh all prices** or **Refresh price**).
**Origin:** Issue #204: since Amazon pages are read through a web browser, every tracked Amazon item shows **Retrieval error**.
**Covers:** FR-AMZ-BRET-001, FR-AMZ-BRET-002, FR-AMZ-BRET-003, FR-AMZ-BRET-004

## Preconditions

- The server is a documented installation (FR-AMZ-BRET-006).
- An Amazon item is tracked and not stopped; its listing is an available listing of a supported marketplace.

## Main flow

1. The application opens the item's Amazon product page in the browser, without showing it and without any user action or Amazon login.
2. Amazon shows the product page, possibly with a cookie or consent banner over it.
3. The application reads the page as a product page, even when its texts, links or scripts contain words such as "captcha" or "robot" (FR-AMZ-BRET-003).
4. The application records the price in euros, title and image shown by the page, as before the browser change (FR-AMZ-BRET-001).
5. The application records the second-hand offer result as before the browser change: the latest offer sold by Amazon with its condition when the page or its used offers show one, otherwise no offer (FR-AMZ-BRET-002).
6. The consecutive failure count is reset and no failure is logged or counted.
7. The user sees the item's price and check time in the tracked-items list and on the item details page; the status is not **Retrieval error** and the latest attempt has no error message.

## Alternatives and errors

- **Robot check, non-success response or no answer:** the check fails and the item shows **Retrieval error**; the failure is logged (UC-AMZ-BRET-002), counted and, at the fifth consecutive failure, every tracked Amazon item is stopped (FR-AMZ-BRET-004, FR-AMZ-HCHK-003, FR-AMZ-STOP-001–010). The last successful price is kept (FR-07).
- **Listing really unavailable** (404/410 or Amazon's unavailable text): the item shows **Listing unavailable**, not Retrieval error.
- **Only an instalment amount shown:** the check gives "price not found" (FR-AMAZON-PRICE-003).
- **Item stopped:** no scheduled check runs; see UC-AMZ-BRET-004.
- **Protected mode:** the same applies to the Amazon items of all users.

## Postconditions

- The item's latest check holds the listing's price, title, image and second-hand offer result.
- Item states, labels, messages, schedule and stop rules are unchanged.

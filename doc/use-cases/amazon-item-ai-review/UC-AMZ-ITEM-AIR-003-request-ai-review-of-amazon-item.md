# UC-AMZ-ITEM-AIR-003 — Request an AI review of an Amazon item

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Claude, SQLite store
**Trigger:** The operator chooses **Refresh AI review** on an Amazon item details page.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-005, FR-AMZ-ITEM-AIR-010, FR-AMZ-ITEM-AIR-011, FR-AMZ-ITEM-AIR-012, FR-AMZ-ITEM-AIR-013, FR-AMZ-ITEM-AIR-017, FR-AMZ-ITEM-AIR-018, FR-AMZ-ITEM-AIR-019, FR-AMZ-ITEM-AIR-020, FR-AMZ-ITEM-AIR-021, FR-AMZ-ITEM-AIR-023, FR-AMZ-ITEM-AIR-026

## Preconditions

- The owner has saved a Claude token.
- The Amazon item is not stopped and has a current price (see the definition in the specification).
- No review is running for the item.

## Main flow

1. The operator opens the item details page. The **AI review** section, with its heading, shows the latest review or that no review exists yet, and an enabled **Refresh AI review** button.
2. The operator chooses **Refresh AI review**, with the mouse or the keyboard.
3. The button is disabled and the section shows and announces that a review is in progress.
4. The application sends Claude the data recorded by the item's checks: title, marketplace, current price and price history; the Amazon-sold second-hand offer (price and condition) and its history; the customer rating and number of ratings; and the purchase goal when the item has one. No Amazon request is made.
5. Claude returns a review in English.
6. The section shows and announces the new review at the top of the history, with its date/time and labelled with the price it was based on (for example "Based on price 129,99 €"). The review shows, labelled and in text:
   - the **Price** rating (Good deal / Fair / Overpriced) with its explanation;
   - the estimated fair price range;
   - the **Overall recommendation** (Buy / Wait for a lower price / Avoid) with its explanation.
7. The operator opens the previous reviews, newest first, each with its date/time and reviewed price and without the goal.

## Alternatives and errors

- **No second-hand offer sold by Amazon:** the request states that there is none; the review is made without it.
- **No customer rating on the page, or never recorded:** the request states that the rating is unknown (UC-AMZ-ITEM-AIR-008).
- **No goal:** no goal is sent.
- **Latest check failed after an earlier price:** the item keeps its current price; the review uses the data of its latest successful checks.
- **Repeated clicks while the review runs:** the button is disabled; at most one review runs for the item.
- **Review fails:** UC-AMZ-ITEM-AIR-004.
- **No Claude token:** UC-AMZ-ITEM-AIR-005.
- **Item stopped:** UC-AMZ-ITEM-AIR-006.
- **No current price:** UC-AMZ-ITEM-AIR-007.
- **Token replaced or removed during the review:** the running review finishes or fails with the token it started with.
- **Phone width:** the section is readable without horizontal scrolling.
- **LeBoncoin item:** its review, goal and triggers are unchanged.

## Postconditions

- The new review is stored and shown first in the history; earlier reviews are kept.
- No Amazon request was sent: the price history, the item's check schedule and the consecutive Amazon failure count are unchanged.

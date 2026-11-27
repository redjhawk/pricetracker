# UC-AMZ-ITEM-AIR-011 — Delete a reviewed Amazon item

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, SQLite store
**Trigger:** The operator deletes an Amazon item that has a goal and reviews.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-016

## Preconditions

- The Amazon item has a purchase goal and at least one review.

## Main flow

1. The operator deletes the item as in UC-IM-02 and confirms.
2. The item is deleted with its price history, second-hand offer history, customer rating, goal and all its reviews.

## Alternatives and errors

- **Deletion cancelled or failed:** the item, its goal and its reviews are kept (UC-IM-02).

## Postconditions

- No review or goal of the item remains.

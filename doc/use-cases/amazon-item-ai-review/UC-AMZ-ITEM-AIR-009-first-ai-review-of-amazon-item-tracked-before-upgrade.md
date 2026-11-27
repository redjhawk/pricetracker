# UC-AMZ-ITEM-AIR-009 — Get the first AI review of an Amazon item tracked before the upgrade

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Claude, SQLite store
**Trigger:** After the upgrade, the operator opens an Amazon item that was tracked before this feature.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-007, FR-AMZ-ITEM-AIR-024, FR-AMZ-ITEM-AIR-031, FR-AMZ-ITEM-AIR-022

## Preconditions

- The Amazon item was added before the upgrade, is not stopped and has a current price.
- The owner has a Claude token.

## Main flow

1. The operator opens the item details page.
2. The goal field is empty and the **AI review** section shows that no review exists yet. No review was started by the upgrade.
3. The operator chooses **Refresh AI review**.
4. The review runs as in UC-AMZ-ITEM-AIR-003. When no check has recorded the customer rating since the upgrade, the rating is sent as absent.
5. The section shows the item's first review.

## Alternatives and errors

- **Item stopped or without a current price:** UC-AMZ-ITEM-AIR-006 or UC-AMZ-ITEM-AIR-007.
- **Reviews deleted by issue #111:** they are not restored.

## Postconditions

- The item has its first review; later reviews are produced only by **Refresh AI review**.

# UC-GOAL-CHOICE-008 — View the goal of an item shared by another user

**Primary actor:** Logged-in user (protected mode)
**Supporting systems:** PriceFollower application, LeBoncoin item details page
**Trigger:** The user opens the details page of a LeBoncoin item another user shared with them.
**Origin:** Issue #304 comments: goals of items shared by other users are not offered.
**Covers:** FR-GOAL-CHOICE-011

## Preconditions

- Another user shares with the actor a LeBoncoin item with goal "Office" (FR-SHARE-004).
- The actor's own LeBoncoin items have the goal "Gaming".

## Main flow

1. The actor opens the shared item's details page.
2. The page shows the goal "Office" read-only, as before this feature; no drop-down and no save action are offered.

## Alternatives and errors

- **Shared item without goal:** the page shows no goal and offers no way to set one.
- **Actor's own goal choice:** in the add-item form or on the actor's own items, "Office" is not offered (FR-GOAL-CHOICE-001, UC-GOAL-CHOICE-001).

## Postconditions

- Nothing changed; the shared item's goal stays under its owner's control.

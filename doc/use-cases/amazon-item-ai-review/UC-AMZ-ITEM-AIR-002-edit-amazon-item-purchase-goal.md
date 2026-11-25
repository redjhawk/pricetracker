# UC-AMZ-ITEM-AIR-002 — Edit the purchase goal of an Amazon item

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, SQLite store
**Trigger:** The operator changes or clears the purchase goal on an Amazon item details page and saves it.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-002, FR-AMZ-ITEM-AIR-003, FR-AMZ-ITEM-AIR-004, FR-AMZ-ITEM-AIR-006, FR-AMZ-ITEM-AIR-007, FR-AMZ-ITEM-AIR-022

## Preconditions

- An Amazon item is tracked, with goal A or with no goal (an Amazon item added before this feature has no goal, FR-AMZ-ITEM-AIR-007).

## Main flow

1. The operator opens the item details page. The labelled, multi-line **Purchase goal** field contains goal A, or is empty when the item has no goal.
2. The operator replaces the text with goal B and chooses the save action, with the mouse or the keyboard.
3. Goal B replaces goal A; goal A is not kept.
4. The page announces that the goal is saved.
5. No AI review is started; the **AI review** section is unchanged.
6. After a reload, the field contains goal B. The next **Refresh AI review** sends goal B (UC-AMZ-ITEM-AIR-003).

## Alternatives and errors

- **Goal cleared:** the operator empties the field and saves; after a reload the field is empty and later reviews are requested with no goal.
- **Whitespace-only goal:** it is stored as no goal; after a reload the field is empty (FR-AMZ-ITEM-AIR-004).
- **Unchanged value saved:** nothing changes.
- **Save fails:** the page shows and announces an error, keeps the entered text in the field, and the stored goal and the reviews stay unchanged (FR-AMZ-ITEM-AIR-006).

## Postconditions

- The item has the saved goal; its reviews and their history are unchanged.

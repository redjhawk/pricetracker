# UC-AMZ-ITEM-AIR-001 — Add an Amazon item with a purchase goal

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, Amazon, SQLite store
**Trigger:** The operator adds an Amazon listing with the add-item form and fills in the optional **Purchase goal** field.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-001, FR-AMZ-ITEM-AIR-004, FR-AMZ-ITEM-AIR-010, FR-AMZ-ITEM-AIR-022

## Preconditions

- The operator is signed in and can add items (UC-IM-01).

## Main flow

1. The operator opens the add-item form and enters an Amazon product URL.
2. The operator enters the goal "gift for a 10-year-old" in **Purchase goal**.
3. The operator submits the form.
4. The item is added and its first check runs as for any Amazon item (UC-AMZ-HCHK-002); the goal is saved with the item.
5. No AI review is started.
6. The operator opens the item details page: the goal field contains "gift for a 10-year-old" and the **AI review** section shows that no review exists yet.

## Alternatives and errors

- **Goal left empty or whitespace only:** the item is added with no goal; its details page shows an empty goal field (FR-AMZ-ITEM-AIR-004).
- **Long multi-paragraph goal:** it is saved and shown in full; leading and trailing whitespace may be trimmed (FR-AMZ-ITEM-AIR-004).
- **Adding the item fails:** no item and no goal are saved, as in UC-IM-01.

## Postconditions

- The Amazon item is tracked with its goal and has no review. A review is produced only by **Refresh AI review** (UC-AMZ-ITEM-AIR-003).

# UC-AMZ-ITEM-AIR-005 — Open an Amazon item when no Claude token is saved

**Primary actor:** Operator (in protected mode, the item's owner)
**Supporting systems:** PriceFollower application, SQLite store
**Trigger:** The operator opens an Amazon item details page while the owner has no Claude token.
**Origin:** Issue #305.
**Covers:** FR-AMZ-ITEM-AIR-015, FR-AMZ-ITEM-AIR-022, FR-CLT-SET-014

## Preconditions

- The owner has no Claude token saved.
- The Amazon item is not stopped and has a current price.

## Main flow

1. The operator opens the item details page.
2. The item's prices are shown as usual; price tracking continues without a token.
3. The **AI review** section says "Configure a Claude token in Settings", still shows any earlier reviews, and the **Refresh AI review** button is disabled.
4. The operator opens Settings; the Claude token help text states that the token is used for Amazon and LeBoncoin AI reviews. The operator saves a Claude token.
5. No Amazon item is reviewed automatically; back on the details page, the button is enabled and the shown review is unchanged.

## Alternatives and errors

- **Item stopped:** the stopped message is shown instead and the button is not shown (UC-AMZ-ITEM-AIR-006).
- **Item without a current price:** the no-current-price message is shown instead and the button is not shown (UC-AMZ-ITEM-AIR-007).

## Postconditions

- No review was attempted without a token; reviews start only when the operator chooses **Refresh AI review**.

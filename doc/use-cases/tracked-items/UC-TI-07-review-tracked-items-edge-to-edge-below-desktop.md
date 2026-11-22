# UC-TI-07 — Review tracked items edge to edge below desktop width

**Primary actor:** Operator, regular user or administrator
**Trigger:** The user opens the tracked items list at any viewport narrower than 1056 px (phone, tablet or narrow desktop window), where the items are shown as stacked cards.
**Origin:** Issue #165: the page content padding and the tab panel padding left 48 px unused on each side of the cards, and the cards left their right-hand part empty.
**Covers:** FR-RESPONSIVE-014, FR-RESPONSIVE-015, FR-RESPONSIVE-016, FR-RESPONSIVE-018, FR-RESPONSIVE-019, FR-RESPONSIVE-020, FR-RESPONSIVE-021, FR-RESPONSIVE-022, FR-RESPONSIVE-024

## Preconditions

- Tracked items exist on the Amazon and LeBoncoin tabs, including an Amazon item with a second-hand offer.

## Main flow

1. The user opens the tracked items list on the Amazon tab in a window, for example, 775 px wide.
2. The title, item count, Add item and refresh-all actions, tabs and search field use the full width: each is at most 16 px from the screen edges, and none touches an edge.
3. Each item card spans the whole screen width, edge to edge. No page background shows to the left or right of the cards.
4. Inside each card, the labels, values and row actions keep a small inner padding, so nothing touches the screen edge.
5. The card's values are placed side by side, as many on a line as fit: on the Amazon tab Prices, Amazon second-hand offer and Status share a line. A value moves to a new line only when it does not fit beside the previous ones. Each value stays identified by its label or column name.
6. The user switches to the LeBoncoin tab and sees the same edge-to-edge cards, with Prices and Status on the same line.
7. The user reads every value and uses the row actions without any sideways scrolling.

## Alternatives and errors

- **Phone width (320 px to 671 px):** the cards are also edge to edge (UC-TI-06). Values that do not fit side by side continue on the next line and stay visible, identifiable and operable, in visual reading order.
- **Search with no match:** the user types a text that matches no item. "No items match your search." is shown inside the full-width list, inset from both screen edges like card content.
- **No item on the tab, loading, or Amazon-stop warning or error notification shown:** the empty state, loading indicator, warning and notification are at most 16 px from each screen edge and do not touch it.
- **Resize across 1056 px:** at 1055 px the list shows edge-to-edge cards; at 1056 px it shows the desktop table with column headers and its current desktop margins. Both switch together, and the selected tab and search text are kept.
- **Desktop (1056 px or more):** the layout is unchanged (UC-TI-01).

## Postconditions

- The cards run edge to edge at every width below 1056 px, their content stays inset and uses the card width, and the page does not scroll sideways.

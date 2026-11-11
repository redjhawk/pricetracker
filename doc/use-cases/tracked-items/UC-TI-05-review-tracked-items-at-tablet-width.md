# UC-TI-05 — Review tracked items at tablet width

**Primary actor:** Operator, regular user or administrator
**Trigger:** The user opens the tracked items list (or the administrator's users list) in a window between 672 px and 1055 px wide, such as a tablet or a narrow desktop window.
**Origin:** QA finding QA-F001 (issue #140). At these widths the table overflowed inside Carbon's inner table scroller. On the Amazon tab, and on LeBoncoin at 672–800 px, the Status column and the row actions were cut off and could only be reached by scrolling the table sideways.

## Preconditions

- Tracked items exist, including long titles and an Amazon item with a second-hand offer.

## Main flow

1. The user opens the list on the Amazon or LeBoncoin tab.
2. Each item is shown as a stacked card with its title, listing ID, and the labelled values "Prices", "Amazon second-hand offer" (Amazon tab) and "Status", followed by the row actions.
3. The user reads every value and opens the row actions without any sideways scrolling, either of the page or inside the table.

## Alternatives and errors

- In the administrator's users list, each user's username and last login are stacked the same way.
- At 1056 px or more, the regular table with column headers is shown, unchanged.

## Postconditions

- No information or action is hidden horizontally (FR-RESPONSIVE-001 to 004).

Regression test: `QA-F001 <width>px tables do not scroll inside and keep Status and actions on screen` (700, 800 and 1000 px) in `tests/responsive-layout.spec.ts`.

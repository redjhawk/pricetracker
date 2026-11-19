# Functional specification: responsive layout

Status: ready\
Owner: functional specification agent\
User decision/reference: GitHub issue #140 (owner): “La interfaz no es responsive y si lo veo desde un móvil tengo que hacer scroll de derecha a izquierda para ver toda la información. La información debería verse aunque la pantalla sea pequeña, del tamaño de un móvil. Adapta todas las pantallas: lista, información de un producto, etc.”

## Purpose and scope

Let the operator use every screen of the interface from a phone without scrolling the page sideways, while seeing the same information and actions as on a desktop. Actors: anyone using the interface (operator in open mode, regular user, administrator).

Reference viewports: **phone** = 360 CSS px wide (reference) and 320 CSS px wide (minimum supported); **desktop** = 1056 CSS px wide or more.

Included screens (all existing screens in `src/`):

- Application header and header menu (Settings, Information, Log in / Log out entries).
- Login page.
- Tracked items page: title, Add item and refresh-all actions, item count, search, Amazon / LeBoncoin tabs, Amazon-stop warning, empty states, and the tracked-items table (Item, Prices, Amazon second-hand offer, Status, actions).
- Item details page: metadata, status, source link, refresh and delete actions, price and second-hand detection history, purchase goal, AI review section and history.
- Administrator page: users table (Username, Last login) and Add user action.
- Modals: Add item, Delete item, Settings, Information, Add user (administrator), and any purchase-goal editing or AI-review controls.

Excluded: new features, new information, removing or hiding information on small screens, changing texts, behavior, data, or the desktop layout; native mobile apps; landscape-specific or tablet-specific designs beyond the general rule below.

## Requirements

| ID | Trigger / precondition | Required behavior | Observable acceptance criteria |
| --- | --- | --- | --- |
| FR-RESPONSIVE-001 | Any screen or modal is shown at any viewport width from 320 px up. | The page as a whole never needs horizontal scrolling. | Given a 360 px and a 320 px viewport, when each included screen and modal is opened (with realistic data, see FR-RESPONSIVE-010), then the document is not wider than the viewport and no sideways page scroll is possible. |
| FR-RESPONSIVE-002 | Phone viewport. | All information shown on desktop for that screen is also shown, readable, without requiring sideways scrolling of any area. | Given the same data at 360 px and at desktop width, every label and value visible on desktop (titles, marketplace and listing ID, prices and price periods, second-hand offer and condition, status, dates, usernames, messages) is present at 360 px and fully visible by vertical scrolling only; no text is clipped or overlaps other content. |
| FR-RESPONSIVE-003 | Phone viewport. | All actions available on desktop remain available and operable. | At 360 px and 320 px, every button, link, tab, menu entry, input, and row action available on desktop can be seen and activated by touch, without zooming or sideways scrolling. |
| FR-RESPONSIVE-004 | Tracked items page, phone viewport. | Each tracked item's Item, Prices, (Amazon tab) Amazon second-hand offer, Status, and row actions are presented within the screen width; the layout may change from a wide table to a stacked presentation, but each value stays identifiable by its label or column name. | At 360 px with Amazon and LeBoncoin items, every item shows all the values listed, each associated with its label (“Prices”, “Amazon second-hand offer”, “Status”), and opening the item and its other row actions work as on desktop. |
| FR-RESPONSIVE-005 | Tracked items page, phone viewport. | The header area (title, item count, Add item, refresh all, search) and the Amazon / LeBoncoin tabs fit the screen width; controls may wrap onto several lines. | At 320 px, both tabs, the search field, and both actions are fully visible and usable; the search keeps filtering the selected tab. The Amazon-stop warning and empty states wrap within the width. |
| FR-RESPONSIVE-006 | Item details page, phone viewport. | Metadata, actions, price and second-hand detection histories, purchase goal, and the AI review section with its history fit the screen width. | At 360 px, all history entries (date, price, condition where applicable) and all AI review fields are fully readable by vertical scrolling only; refresh, delete, source-link, purchase-goal and re-review actions are usable. |
| FR-RESPONSIVE-007 | Administrator page, phone viewport. | The users list and the Add user action fit the screen width. | At 320 px, each user's username and last login are fully visible and Add user is usable. |
| FR-RESPONSIVE-008 | Any modal open on a phone viewport. | The modal fits within the screen width; its content, inputs, notifications, and footer buttons are visible and usable; long content scrolls vertically inside the modal. | At 320 px, Add item, Delete, Settings, Information, and Add user modals show all their fields, messages, and buttons without sideways scrolling, and can be submitted and closed as on desktop. |
| FR-RESPONSIVE-009 | Header and login page on a phone viewport. | The header (application name, menu icon, Log in link in open mode) and the login form fit the screen width; the header menu opens fully on screen. | At 320 px, the header items do not overlap or overflow; opening the menu shows all its entries within the screen; the login form fields and button are fully visible and usable. |
| FR-RESPONSIVE-010 | Long content: long item titles, long URLs, long usernames, long purchase goals, long AI review text, long error messages, prices with many digits. | Long content wraps (or breaks within words/URLs when necessary) instead of widening the page. | With a 200-character title without spaces and a long listing URL at 320 px, the page does not scroll sideways and the full text remains readable. |
| FR-RESPONSIVE-011 | Desktop viewport (1056 px or more). | Layout and behavior are unchanged from the current release. | At desktop width every screen and modal looks and behaves as before this change (same table columns, positions, and actions). |
| FR-RESPONSIVE-012 | Phone viewport, keyboard or assistive technology. | Existing accessibility is preserved: reading order follows the visual order, labels stay associated with values, focus is visible, and touch targets keep Carbon's sizes. | At 360 px, screen readers announce each value with its label; Tab order follows the visual order; focus indicators are visible. |
| FR-RESPONSIVE-013 | The viewport width changes (rotation or window resize) while a screen or modal is open. | The layout adapts without losing entered data, the selected tab, the search query, or the open modal. | Rotating a phone from portrait to landscape and back keeps the tab, search text, and modal inputs. |
| FR-RESPONSIVE-014 | Tracked items page, phone viewport (≤ 671 px; reference 360 px and 320 px), Amazon or LeBoncoin tab, at least one item listed. | The tracked-items list (the stacked item cards, plus any element inside the same list container such as its search toolbar row) spans the full viewport width, edge to edge, with no lateral page margin on the left or right. | At 360 px and 320 px, on both tabs, each item card's left edge is at x = 0 and its right edge at the viewport width; no page background is visible beside the cards. |
| FR-RESPONSIVE-015 | Same as FR-RESPONSIVE-014. | The content inside each full-width card keeps a small inner padding so that no text, value, or action touches the screen edge. | At 320 px, every label, value, and row action in each card is inset from both screen edges and fully readable and operable (FR-RESPONSIVE-002/003 still hold). |
| FR-RESPONSIVE-016 | Same as FR-RESPONSIVE-014. | Making the list full width introduces no horizontal page scroll (FR-RESPONSIVE-001 still applies), and elements above the list (title, item count, Add item, refresh all, tabs, Amazon-stop warning) and empty states keep their current layout and margins. | At 320 px the document is not wider than the viewport; the header area and tabs look as before this change. |
| FR-RESPONSIVE-017 | Desktop viewports (≥ 1056 px), and every other screen and modal at any width. | Unchanged by FR-RESPONSIVE-014 to 016 and FR-RESPONSIVE-018 to 021. (Issue #165 narrowed this from “≥ 672 px” to “≥ 1056 px”.) | At 1056 px and wider, the tracked-items list keeps its current margins; item details, administrator, login, header, and modals look as before these changes at every width. |
| FR-RESPONSIVE-018 | Tracked items page, any viewport width at which the tracked items are shown as stacked cards instead of a table with column headers (today every width below 1056 px, i.e. phone and tablet), Amazon or LeBoncoin tab, at least one item listed. | The tracked-items list spans the full viewport width, edge to edge: neither the page content margin nor the tab panel margin is left on the left or right of the cards. FR-RESPONSIVE-014 thus applies to the whole card range, not only to phones. | At 320 px, 360 px, 671 px, 672 px, 775 px, and 1055 px, on both tabs, each item card's left edge is at x = 0 and its right edge at the viewport width; no page background is visible beside the cards. |
| FR-RESPONSIVE-019 | Same as FR-RESPONSIVE-018. | The full-width list and the card layout start and stop at the same viewport width: there is no width at which cards are shown with lateral margins, and no width at which the table with column headers is shown edge to edge. | At 1055 px the list shows edge-to-edge cards; at 1056 px it shows the table with column headers and its current desktop margins (FR-RESPONSIVE-011). Resizing across that width switches both together. |
| FR-RESPONSIVE-020 | Same as FR-RESPONSIVE-018. | FR-RESPONSIVE-015 (small inner padding inside each card) and FR-RESPONSIVE-016 (no horizontal page scroll; elements outside the list keep their current layout and margins) apply over the whole card range. | At 320 px, 775 px, and 1055 px, every label, value, and row action in each card is inset from the screen edges and readable; the document is not wider than the viewport; the title, item count, Add item, refresh all, tabs, search field, Amazon-stop warning, error notification, and empty states look as before this change. |
| FR-RESPONSIVE-021 | Same as FR-RESPONSIVE-018, search with no matching item. | The “No items match your search.” message is shown inside the full-width list, inset from the screen edges like card content. | At 360 px and 775 px, searching for a text that matches no item shows the message fully readable and not touching either screen edge; no horizontal page scroll. |

## States and corner cases

- Loading, empty, error, and pending states (including inline notifications and loading indicators) follow FR-RESPONSIVE-001/002/010 on every screen.
- Many items (about 100) only extend the page vertically.
- Browser zoom and text size are not changed by the application (no zoom blocking).
- Widths between 320 px and desktop must also satisfy FR-RESPONSIVE-001 to 003; intermediate layouts are a technical choice, except that the tracked-items list follows FR-RESPONSIVE-018 to 021 wherever it is shown as cards.
- Below 320 px is not supported.

## Open questions

None. The request explicitly covers all screens and requires all information to be visible on phone-sized screens; reference widths are a testing convention, not a functional choice.

## Traceability

- Issue #140; change record [index](../../changes/2026-11-11-1142-issue-140-responsive-layout/index.md).
- Follow-up (owner, issue #140): “implementa que la lista ocupe todo el ancho en móvil, sin márgenes laterales” → FR-RESPONSIVE-014 to 017; change record [index](../../changes/2026-11-11-1513-issue-140-full-width-list/index.md).
- Issue #165 (owner): “Use all space to show item list when the screen is reduced”; owner comment: the page content padding and the tab panel padding (48 px per side) leave unused space on small screens, also below 672 px where cards were meant to be edge to edge; “padding must disappear as soon as the inside card layout change[s]” → FR-RESPONSIVE-018 to 021, FR-RESPONSIVE-017 narrowed to ≥ 1056 px. Scope is the tracked-items list named by the issue; other screens are unchanged. Change folder: [issue 165](../../changes/2026-11-19-2053-issue-165-use-all-space-to-show-item-list-when-the/issue.md).
- Related: [platform tabs](../platform-tabs/functional.md), [tracked items identity](../tracked-items-identity/functional.md), [header menu](../app-header-menu/functional.md), [application information](../app-information/functional.md) FR-APP-INFO-013, [multi-user login](../multi-user-login/functional.md), [LeBoncoin AI review](../leboncoin-ai-review/functional.md), [purchase goal](../leboncoin-purchase-goal/functional.md), [Amazon request stop](../amazon-request-stop/functional.md).
- Technical handoff target: `doc/specifications/responsive-layout/technical.md`.

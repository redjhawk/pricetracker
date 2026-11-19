# UC-RL-01 — Use the full screen width on every page below desktop width

**Primary actor:** Operator, regular user or administrator
**Trigger:** The user opens any page (tracked items, item details, administrator, login) at a viewport narrower than 1056 px.
**Origin:** Issue #165, owner answer: "all screen should use available space".
**Covers:** FR-RESPONSIVE-001, FR-RESPONSIVE-016, FR-RESPONSIVE-017, FR-RESPONSIVE-022

## Preconditions

- The application has data to show: tracked Amazon and LeBoncoin items (a LeBoncoin item with a purchase goal and AI review history) and, for the administrator page, at least one user.

## Main flow

1. The user opens the item details page of a LeBoncoin item in a window 775 px wide.
2. The page has no lateral page margins: the metadata, actions, price history, purchase goal and AI review section with its history are each at most 16 px from the screen edges. No text or control touches an edge.
3. The user opens the item details page of an Amazon item; its second-hand detection history follows the same rule.
4. The user opens the tracked items page; it follows the same rule (UC-TI-07).
5. The administrator opens the administrator page; the title, Add user action and users list follow the same rule (UC-RL-02).
6. The user, logged out, opens the login page. The login form's fields and button span the screen width minus at most 16 px on each side.
7. On every page the document is never wider than the viewport, so the page never scrolls sideways.

## Alternatives and errors

- **Loading, empty or error state:** the loading indicator, empty-state message or error notification follows the same rule: at most 16 px from each screen edge, never touching it.
- **Phone width (320 px to 671 px):** the same rule applies; long content wraps instead of widening the page.
- **Desktop (1056 px or more):** every page keeps its current margins and layout.
- **Modal or application header:** modals (Add item, Delete item, Settings, Information, Add user, purchase goal, AI review) and the header look as before at every width.

## Postconditions

- Below 1056 px every page uses the available screen width; the desktop layout, modals and header are unchanged.

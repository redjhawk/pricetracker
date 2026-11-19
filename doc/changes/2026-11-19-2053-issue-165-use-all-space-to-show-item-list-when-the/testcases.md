# Test cases — issue #165: use all space below desktop width

Scope: [responsive layout specification](../../specifications/responsive-layout/functional.md) FR-RESPONSIVE-014 to 024 (new or changed by issue #165), and the existing requirements they must keep (001 to 013). Use cases: UC-TI-05, UC-TI-06, UC-TI-07, UC-RL-01, UC-RL-02.

## Levels

The change is presentation only: no API, store or handler behaviour changes. Go `unit` and `integration` levels have nothing to prove here, so every case is `e2e` (Playwright, `tests/responsive-layout.spec.ts`), using the existing mocked API with synthetic data.

## Shared definitions

- **Widths.** Phone 320 and 360 px; card range boundaries 671, 672 and 1055 px; the issue's screenshot width 775 px; desktop 1056 and 1440 px.
- **Edge gap.** For each screen edge, the distance from that edge to the nearest page content: a text range, a control, or the outer box of a filled or bordered element (card, tile, list, notification, form field, tab bar). The application header and open modals are excluded.
- **Touches an edge.** A text range or control whose left is ≤ 0 or whose right is ≥ the viewport width.
- **No horizontal scroll.** `document.documentElement.scrollWidth <= clientWidth` (existing `expectNoHorizontalScroll`).
- **Edge to edge.** Card or list box with |left| ≤ 1 px and |right − viewport width| ≤ 1 px.
- **Inset.** Text and controls inside a card have left ≥ 4 px and right ≤ viewport width − 4 px, and they do not touch an edge.
- **Same line.** Two values whose boxes overlap vertically (the top of one is above the bottom of the other).
- **Sample-like data.** The mocked items use values shaped like the development seed data: an Amazon item with a short title (for example “Sony WH-1000XM5 Wireless Noise Canceling Headphones”), three prices, an available second-hand offer and a success status; a LeBoncoin item with a short title and prices; a user with a short username. The existing long-content items (200-character title, long URLs, 80-character username) are used for the overflow cases.
- **Baseline.** Positions measured on `master` before this change (page content offsets, header items, modal boxes) and recorded as constants in the test.

## Cases

### TC-001 — Tracked-item cards run edge to edge over the whole card range

- **Covers:** FR-RESPONSIVE-014, FR-RESPONSIVE-018, FR-RESPONSIVE-001; UC-TI-07, UC-TI-06, UC-TI-05.
- **Level:** e2e.
- **Given** the tracked items page with Amazon and LeBoncoin items, including the long-content items.
- **When** it is shown at 320, 360, 671, 672, 700, 775 and 1055 px, first on the Amazon tab and then on the LeBoncoin tab.
- **Then** each item card is edge to edge, as is the list container. No page background is visible beside the cards, and there is no horizontal scroll.
- **Corner cases:** both breakpoint sides (671 and 672 px); the last card width (1055 px); a 200-character title without spaces and a long URL; tab switching.

### TC-002 — Card content stays inset and readable

- **Covers:** FR-RESPONSIVE-015, FR-RESPONSIVE-020, FR-RESPONSIVE-002, FR-RESPONSIVE-003, FR-RESPONSIVE-004; UC-TI-07, UC-TI-06.
- **Level:** e2e.
- **Given** the tracked items page with both item kinds.
- **When** it is shown at 320, 775 and 1055 px on both tabs.
- **Then** every label, value, item title button and Options button in each card is inset and visible. The labels “Prices”, “Amazon second-hand offer” (Amazon tab only) and “Status” are present in each card. Opening the item and its Options menu works.
- **Corner cases:** the narrowest width (320 px); long titles that wrap; a LeBoncoin card without a second-hand value.

### TC-003 — The card layout and the full width start and stop together at 1056 px

- **Covers:** FR-RESPONSIVE-019, FR-RESPONSIVE-011, FR-RESPONSIVE-013, FR-RESPONSIVE-017; UC-TI-07 (resize across 1056 px).
- **Level:** e2e.
- **Given** the tracked items page on the LeBoncoin tab, with search text “bicycle”.
- **When** the viewport is 1055 px, then 1056 px, then 1055 px, then 360 px, then 775 px.
- **Then:**
  - At 1055, 775 and 360 px, the table header row is not displayed and the cards are edge to edge.
  - At 1056 px, the column headers (“Prices”, “Status”) are visible, and the table's left and right offsets equal the desktop baseline (greater than 16 px).
  - Neither mixed state ever appears: no cards with lateral margins, and no edge-to-edge table with column headers.
  - The LeBoncoin tab stays selected, and the search keeps “bicycle” and the filtered result.
- **Corner cases:** repeated resizing across the boundary; resizing within the card range.

### TC-004 — “No items match your search.” shown inside the full-width list

- **Covers:** FR-RESPONSIVE-021, FR-RESPONSIVE-016; UC-TI-07 (search with no match).
- **Level:** e2e.
- **Given** the tracked items page with items.
- **When** “nothing-matches” is typed in the search field, at 360 and at 775 px, on both tabs.
- **Then** “No items match your search.” is visible inside the list container, which is edge to edge. The message text is inset, does not touch either edge, and is at most 16 px from each edge's nearest content box. There is no horizontal scroll.
- **Corner cases:** empty result after filtering; clearing the search brings back edge-to-edge cards.

### TC-005 — Card values placed side by side, wrapping only when needed

- **Covers:** FR-RESPONSIVE-024; UC-TI-07 (main flow step 5), UC-TI-05.
- **Level:** e2e.
- **Given** the tracked items page with sample-like data.
- **When** it is shown at 775 and 1055 px.
- **Then:**
  - On the Amazon tab, each card's Prices, Amazon second-hand offer and Status values are on the same line.
  - On the LeBoncoin tab, Prices and Status are on the same line.
  - For every value that starts a later line in a card, the free space to the right of the previous line is smaller than that value's width, so the right part of the card is not left empty while a value would fit there.
  - Each value is still associated with its label or column name.
- **Corner cases:** the two widths in the card range; both tabs; an Amazon card with and one without a second-hand offer.

### TC-006 — Card values at phone width stay visible, identifiable, operable and in reading order

- **Covers:** FR-RESPONSIVE-024, FR-RESPONSIVE-004, FR-RESPONSIVE-003, FR-RESPONSIVE-012; UC-TI-07 and UC-TI-06 (phone alternative).
- **Level:** e2e.
- **Given** the tracked items page with the long-content and sample-like items.
- **When** it is shown at 320 px, the user reads each card, and then the user presses Tab repeatedly from the search field.
- **Then:**
  - Every value and row action is visible, not clipped and labelled.
  - The card's values in DOM order match their visual order (top to bottom, then left to right).
  - The keyboard focus moves through each card's item title and Options button in visual order, and the focused element is fully inside the viewport with a visible focus indicator.
  - Pressing Enter on the Options button opens the row menu within the screen.
- **Corner cases:** values wrapping onto several lines; keyboard-only use; no horizontal scroll at 320 px.

### TC-007 — Elements above the tracked-items list use the full width

- **Covers:** FR-RESPONSIVE-022, FR-RESPONSIVE-016, FR-RESPONSIVE-020, FR-RESPONSIVE-005; UC-TI-07 (main flow step 2), UC-RL-01, UC-TI-06.
- **Level:** e2e.
- **Given** the tracked items page with items and the Amazon-stop warning (`amazonStoppedAt` set).
- **When** it is shown at 360, 775 and 1055 px.
- **Then** each of the title, item count, Add item, Refresh all prices, the Amazon / LeBoncoin tab bar, the search field and the Amazon-stop warning is checked:
  - its left and right edge gaps are ≤ 16 px for full-width blocks (tab bar, search field, warning), and it is the nearest content to the edge at ≤ 16 px for wrapped heading rows;
  - no text or control touches an edge;
  - the search still filters the selected tab.

  There is no horizontal scroll.
- **Corner cases:** controls wrapping onto several lines at 360 px; the warning's long text wraps.

### TC-008 — Tracked items page states (empty, loading, error) use the full width

- **Covers:** FR-RESPONSIVE-022, FR-RESPONSIVE-020; UC-TI-07 and UC-RL-01 (loading, empty or error state).
- **Level:** e2e.
- **Given** three states: an items API returning no items; an items API held unanswered (loading); and an items API returning HTTP 500.
- **When** the tracked items page is shown in each state at 360 and 775 px.
- **Then** the empty-state message (“No items tracked yet”), the loading indicator and the error notification each have edge gaps ≤ 16 px on both sides and do not touch an edge. There is no horizontal scroll.
- **Corner cases:** no data; slow request; server error.

### TC-009 — Item details page uses the full width

- **Covers:** FR-RESPONSIVE-022, FR-RESPONSIVE-006, FR-RESPONSIVE-010, FR-RESPONSIVE-001; UC-RL-01 (main flow steps 1–3, loading/error alternative).
- **Level:** e2e.
- **Given** an Amazon item with second-hand detection history, and a LeBoncoin item with a 200-character purchase goal and AI review history.
- **When** each item details page is shown at 360, 775 and 1055 px, and also in a loading state (request held) and an error state (HTTP 500 / unknown item) at 775 px.
- **Then:**
  - The metadata, the actions (Refresh price, Delete, source link), the price and second-hand histories, the purchase goal and the AI review section with its history each have edge gaps ≤ 16 px on both sides.
  - No text or control touches an edge.
  - The long purchase goal and long URL wrap, and there is no horizontal scroll.
  - The loading indicator and the error notification follow the same rule.
- **Corner cases:** long content; both platforms; loading and error states.

### TC-010 — Administrator page: users list edge to edge, values side by side

- **Covers:** FR-RESPONSIVE-023, FR-RESPONSIVE-024, FR-RESPONSIVE-022, FR-RESPONSIVE-007; UC-RL-02, UC-RL-01 (step 5).
- **Level:** e2e.
- **Given** a logged-in administrator. Run 1 has one user with a short username. Run 2 has one user with an 80-character username.
- **When** the administrator page is shown at 360, 775 and 1055 px (run 1) and at 320 px (run 2).
- **Then:**
  - Run 1:
    - Each user card is edge to edge.
    - Username and Last login are inset, labelled and fully readable.
    - At 775 and 1055 px they are on the same line.
    - The page title and the Add user button have edge gaps ≤ 16 px and do not touch an edge.
    - Add user opens its dialog.
  - Run 2 (320 px):
    - The long username wraps inside the card.
    - Last login is still visible, either on the same line or on the next one, and is labelled.
    - There is no horizontal scroll.
- **Corner cases:** long username; narrowest width; Last login moving to the next line only when it does not fit.

### TC-011 — Login page uses the full width

- **Covers:** FR-RESPONSIVE-022, FR-RESPONSIVE-009; UC-RL-01 (main flow step 6).
- **Level:** e2e.
- **Given** protected mode with no logged-in user.
- **When** the login page is shown at 360, 775 and 1055 px, then a login is submitted with wrong credentials (the login API returns 401) at 775 px.
- **Then:**
  - The username field, the password field and the Log in button are within 16 px of each side of the screen; at 775 px they span the screen width minus at most 16 px on each side.
  - No text or control touches an edge.
  - The error notification follows the same rule.
  - There is no horizontal scroll.
- **Corner cases:** error state; the widest card-range width (1055 px).

### TC-012 — Desktop layout unchanged

- **Covers:** FR-RESPONSIVE-011, FR-RESPONSIVE-017, FR-RESPONSIVE-019; UC-TI-07, UC-RL-01, UC-RL-02 (desktop alternative).
- **Level:** e2e.
- **Given** the same data as TC-001, TC-009, TC-010 and TC-011.
- **When** the tracked items page (both tabs), the item details pages, the administrator page and the login page are shown at 1056 and 1440 px.
- **Then:**
  - The page content left and right offsets equal the baseline.
  - The tracked-items and users tables show their column headers.
  - The tracked table lines up with the search row, as before.
  - There is no horizontal scroll.
- **Corner cases:** exactly 1056 px; a wide desktop.

### TC-013 — Header and modals unchanged below desktop width

- **Covers:** FR-RESPONSIVE-017, FR-RESPONSIVE-008, FR-RESPONSIVE-009; UC-RL-01 (modal or application header alternative).
- **Level:** e2e.
- **Given** the tracked items page with items, and the administrator page.
- **When** the following are shown at 360 and 775 px:
  - the header and its open menu;
  - the Add tracked item, Delete tracked item, Settings, Information and Add user modals.
- **Then:**
  - The header items' boxes and each modal container's box equal the baseline.
  - Each modal's footer buttons are in the viewport, and Escape closes the modal.
  - There is no horizontal scroll.
- **Corner cases:** opening and closing modals repeatedly; Escape key.

## Existing tests to update

- `700px tracked-items list keeps its page margins` in `tests/responsive-layout.spec.ts` asserts the behaviour that FR-RESPONSIVE-018 replaces. Remove it; TC-001 (700 px) covers that width.
- `320px` / `360px tracked-items list spans the full width` stay valid. TC-001 and TC-002 extend them.

## Coverage matrix

| Requirement / use case | Test cases |
| --- | --- |
| FR-RESPONSIVE-014 | TC-001 |
| FR-RESPONSIVE-015 | TC-002 |
| FR-RESPONSIVE-016 | TC-004, TC-007 |
| FR-RESPONSIVE-017 | TC-003, TC-012, TC-013 |
| FR-RESPONSIVE-018 | TC-001 |
| FR-RESPONSIVE-019 | TC-003, TC-012 |
| FR-RESPONSIVE-020 | TC-002, TC-007, TC-008 |
| FR-RESPONSIVE-021 | TC-004 |
| FR-RESPONSIVE-022 | TC-004, TC-007, TC-008, TC-009, TC-010, TC-011 |
| FR-RESPONSIVE-023 | TC-010 |
| FR-RESPONSIVE-024 | TC-005, TC-006, TC-010 |
| Kept: FR-RESPONSIVE-001 to 013 | TC-001 to TC-013 (no horizontal scroll, information, actions, accessibility, resize) |
| UC-TI-05 | TC-001, TC-005 |
| UC-TI-06 | TC-001, TC-002, TC-006, TC-007 |
| UC-TI-07 | TC-001 to TC-008 |
| UC-RL-01 | TC-007, TC-008, TC-009, TC-011, TC-012, TC-013 |
| UC-RL-02 | TC-010 |

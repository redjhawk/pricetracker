# QA execution: issue #140 responsive layout

Status: executed — **1 failure (QA-F001)**
Tested code/specification revisions: uncommitted working tree on `ai-dev/issue-140-20261111-1142` (base `f6ed820`); `doc/specifications/responsive-layout/functional.md` and `technical.md` (ready); [decisions.md](decisions.md).
Environment: `npm run dev` (Vite 6.4.3 at `http://localhost:5173`, Go API with `PRICEFOLLOWER_ENV=development` seed data: 8 items, 2 LeBoncoin and 6 Amazon, open mode). Playwright Chromium 153.0.8010.12 headless, Node v22.23.4, throwaway scripts in `.qa-tmp/` (deleted). For the login page (protected mode) and the admin page (admin session plus a user list with a 66-character unbroken username), `GET /api/v1/auth/session` and `/api/v1/admin/users` were intercepted in the browser; everything else used the real local API. No item was created or deleted (modals were opened and closed only).
Executed at: 2026-11-11T12:05Z to 12:20Z (UTC)
Tester: independent QA tester agent

## Exploratory session

Automated, deterministic LCG (`seed = (seed*1103515245+12345) mod 2^31`), seed **140111142**, 40 actions starting at `http://localhost:5173/` at 360 px. Actions drawn from: resize (320, 360, 414, 672, 768, 800, 1000, 1055, 1056, 1280), tab switch, search ("", "zz", "a", "ü€%", 120 × "x"), open a random item, home, browser back, Tab ×5, open/close header menu. After every action `scrollWidth` vs `clientWidth` was measured. Result: 40/40 steps without page overflow. Step 1 (`back`) left the app for `about:blank`, so steps 2–5 were no-ops and step 5 (tab click) timed out; this is an artifact of the script, not an app defect.

## Executed cases

Overflow measurement = `document.documentElement.scrollWidth <= clientWidth`, plus a list of visible elements outside `[0, viewport]`.

| Case | Requirement / corner case | Actual interface URL | Source listing URL | Preconditions and exact actions | Expected | Observed | Result |
| --- | --- | --- | --- | --- | --- | --- | --- |
| QA-001 | FR-RESPONSIVE-001/009 login | `http://localhost:5173/login` | n/a | Protected session mocked; widths 320, 360, 672, 800, 1000, 1055 | No page overflow | scrollWidth = clientWidth at all 6 widths (the only item outside the viewport is Carbon's hidden counter-alert span at -1..0, which is invisible) | Pass |
| QA-002 | FR-001/004/005 list, Amazon tab | `http://localhost:5173/` | n/a | Seed data; click tab "Amazon"; 6 widths | No page overflow; all values/actions visible | No page overflow at any width. At 320/360 the table is stacked, and the only off-screen elements are the visually hidden `thead` cells. **At 672, 800 and 1000 px the table is 959 px wide inside Carbon's `.cds--data-table-content` (`overflow-x: auto`, clientWidth 576/704/904), so the "Amazon second-hand offer", Status and actions columns are cut off and can only be reached by scrolling the table sideways** (screenshot at 800 px: Status and row actions are not visible) | **Fail → QA-F001** |
| QA-003 | FR-001/004/005 list, LeBoncoin tab | `http://localhost:5173/` | n/a | Click tab "LeBoncoin"; 6 widths | Same | No page overflow. **At 672 and 800 px the table (761 px) is wider than the container (576/704 px): Status/actions are clipped behind the inner scroll.** It fits at 1000 and 1055 px. | **Fail → QA-F001** |
| QA-004 | FR-003/005 list controls | `http://localhost:5173/` | n/a | Checked every `button, a, input, [role=tab]` on the LeBoncoin tab for a box outside the viewport | None outside | None at 320, 360, 800, 1000, 1055. At 672, four unnamed row-action buttons are outside (they are inside the clipped table; see QA-F001) | Pass except QA-F001 |
| QA-005 | FR-009 header menu | `http://localhost:5173/` | n/a | Click "Application menu"; 6 widths | All entries on screen | "Information" and "Settings" entries within `[0, width]` (e.g. 160..320 at 320 px); no overflow | Pass |
| QA-006 | FR-008 Settings modal | `http://localhost:5173/` | n/a | Menu → Settings; 6 widths; Escape | Fits | No overflow, no clipped elements | Pass |
| QA-007 | FR-008 Information modal | `http://localhost:5173/` | n/a | Menu → Information; 6 widths | Fits | No overflow | Pass |
| QA-008 | FR-008 Add item modal | `http://localhost:5173/` | n/a | Heading "Add item"; 6 widths | Fits | No overflow | Pass |
| QA-009 | FR-006 details Amazon | `http://localhost:5173/items/sample-lego-land-rover` | n/a | Open the URL; 6 widths | Fits | No overflow, no clipped elements | Pass |
| QA-010 | FR-006 details LeBoncoin | `http://localhost:5173/items/sample-lbc-pool` | n/a | Open the URL; 6 widths | Fits | No overflow | Pass |
| QA-011 | FR-008 Delete modal | the two details URLs above | n/a | Click "Delete" (first match), modal opens; not confirmed | Fits | No overflow at 6 widths, for both items | Pass |
| QA-012 | FR-007 admin page | `http://localhost:5173/admin` | n/a | Admin session and users mocked (66-character username); 6 widths | Fits; username and last login visible | No overflow; long username wraps | Pass |
| QA-013 | FR-008 Add user modal | `http://localhost:5173/admin` | n/a | Click "Add user"; 6 widths | Fits | No overflow | Pass |
| QA-014 | REV-002 / FR-012 accessibility tree | `http://localhost:5173/` and `/admin` | n/a | 360 px; Playwright `ariaSnapshot()` of table "Tracked items" (Amazon, LeBoncoin) and "Users"; CDP `Accessibility.getFullAXTree` | Each labelled value exposed with its label; title/actions named | Chromium keeps table semantics (table 1, rows, cells, columnheaders). Every labelled cell is exposed with its label first, e.g. `cell "Prices 279,00 € 11 Nov, 09:15 …"`, `cell "Amazon second-hand offer 134,99 € Très bon état"`, `cell "Status Active"`, `cell "Username admin"`, `cell "Last login Never"`. Title buttons are named after the item; action buttons are "Open amazon listing in a new tab" and "Options" | Pass (Chromium only) |
| QA-015 | REV-003 / FR-011 desktop unchanged | `/`, `/items/sample-lego-land-rover`, `/items/sample-lbc-pool`, `/admin`, plus Add item/Settings/Information modals | n/a | At 1056 and 1440 px, rendered once with the branch CSS and once with `master:src/index.css` (Vite CSS module response replaced in the browser; the TSX differences are only `className` and `data-label`). Compared the rounded bounding box (x, y, w, h) of every element in `header` and `main` (or in the modal); full-page screenshots taken | Identical | 0 geometry differences on every screen and modal at both widths (e.g. Amazon list: 303/303 elements identical) | Pass |
| QA-016 | FR-013 resize | `http://localhost:5173/` | n/a | 360 px, tab LeBoncoin, search "Best"; resize 1056→360→1056→360; then open Add item, type `https://www.leboncoin.fr/ad/test/42` (not submitted), resize 1056→360 | Tab, search and modal input kept | Tab "LeBoncoin", search "Best", 1 row and no overflow at every step; modal stays open with its input intact | Pass |
| QA-017 | Exploratory | starts at `http://localhost:5173/`; visited `/items/sample-nike-air-max` | n/a | Seed 140111142, 40 actions (see above) | No page overflow | 0 failures | Pass |

## Finding

### QA-F001: table columns cut off at intermediate widths (672 px to about 1006 px)

- Requirement: functional spec "States and corner cases": "Widths between 320 px and desktop must also satisfy FR-RESPONSIVE-001 to 003". FR-RESPONSIVE-002 says "without requiring sideways scrolling of any area"; FR-RESPONSIVE-003 says actions are seen "without … sideways scrolling".
- Reproduction: `npm run dev`, open `http://localhost:5173/` at 800×800, Amazon tab. The table spans 48..1007 px inside `.cds--data-table-content` (`overflow-x: auto`, scrollWidth 959, clientWidth 704). The "Amazon second-hand offer" header is cut off, and the Status column and row actions (open listing, Options) are not visible without scrolling the table area sideways. Same on the LeBoncoin tab at 672 and 800 px (table 761 px). Amazon at 1000 px: 959 > 904.
- Why the automated tests missed it: the 700/1000 px tests from REV-001 check only the document `scrollWidth`. Carbon's inner scroll container hides the overflow from that check.
- Comparison with master: master already used `overflow-x: auto` with `min-width: 66rem`, so these widths also scrolled inside the table area there. This is not a regression, but it fails the requirement for the intermediate range, and it is the situation for which TS-RESPONSIVE-003 gives a fallback (stacked layout up to `max-width: 1055.98px`).
- Proposed handling: adjudication. Per REV-001 ("any overflow QA finds reopens this finding as critical").

## Coverage and limitations

- Covered: all included screens and modals at 6 widths, header menu, empty search result (exploratory), long search input, long unbroken username, keyboard Tab presses (no visible overflow; focus order was not inspected).
- Not covered: real phone touch, VoiceOver/WebKit or another real screen reader (only the Chromium accessibility tree was checked), Firefox/WebKit engines, the purchase-goal edit and AI re-review controls opened as modals, the Amazon-stop warning with live data (the seed has none; the automated spec covers it with fixtures), 200-character title with live data (covered only by the automated spec with fixtures), loading/error states, and submitting modals (avoided to keep the seed data intact).
- Login and admin used intercepted session/user responses in the browser, not a real protected-mode backend.
- The desktop comparison used master's `index.css` with the branch TSX, not a full master build.

## Handoff and cleanup

QA-F001 goes to the review adjudicator (reopens REV-001). If a CSS fix is accepted, retest QA-002, QA-003 and QA-004 at 672/800/1000 px, measuring the `.cds--data-table-content` scrollWidth vs clientWidth, plus QA-015 at 1056 px. Dev server stopped; `.qa-tmp/` (scripts, screenshots, accessibility dumps) deleted; no data mutated.

## Retest after the QA-F001 fix

Status: executed, **pass** (0 failures in 34 checks).
Revision: uncommitted working tree on `ai-dev/issue-140-20261111-1142`, with tables stacked up to `max-width: 1055.98px` (see [decisions.md](decisions.md)).
Environment: same as above (`npm run dev`, `http://localhost:5173`, seed data, Chromium 153.0.8010.12 headless; admin session and users intercepted in the browser).
Executed at: 2026-11-11 about 12:35Z (UTC).

Each check measured page overflow (`documentElement.scrollWidth <= clientWidth`) and any inner scroller (`.cds--data-table-content`, `.cds--data-table-container`, `.tracked-table-container`, `.responsive-table-container`) whose `scrollWidth > clientWidth`. It also counted table cells, links and buttons outside the viewport.

| Case | Actual interface URL | Actions | Observed | Result |
| --- | --- | --- | --- | --- |
| QA-R002 (QA-002) | `http://localhost:5173/` | Tab Amazon, at 672, 800, 1000 and 1055 px | Page width equals the viewport; no inner scroller overflows; 0 cells off screen; the first Status cell and the "Options" row action are inside the viewport; the table is stacked with a label on each value (screenshot at 800 px checked) | Pass |
| QA-R003 (QA-003) | `http://localhost:5173/` | Tab LeBoncoin, same widths | Same as QA-R002 | Pass |
| QA-R004 (QA-004) | `http://localhost:5173/` | Row buttons and links checked against the viewport at the same widths | 0 outside | Pass |
| QA-R012 | `http://localhost:5173/admin` | Admin users table, 700 and 1000 px | No page or inner overflow; 0 cells off screen | Pass |
| QA-R015 (QA-015) | `/`, `/items/sample-lego-land-rover`, `/items/sample-lbc-pool`, `/admin`, plus Add item/Settings/Information modals | 1056 and 1440 px; element geometry compared between the branch CSS and `master:src/index.css` | 0 geometry differences on all 8 screens/modals at both widths; table header row visible on both list tabs and on admin | Pass |
| QA-R-sanity | `/`, `/items/sample-lbc-pool`, Settings modal | 320 and 360 px | No page or inner overflow; 0 cells off screen | Pass |

Observation (not a finding): in the stacked layout at intermediate widths (for example 800 px), the date of each price line is pushed to the right edge, far from its price. It is still readable and fully visible, and no requirement covers it, so it is recorded only for information.

QA-F001: **fixed and retested**. Dev server stopped; `.qa-tmp/` deleted; no data mutated.

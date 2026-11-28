# Technical specification: application header menu

Status: ready (frontend-only subject; implemented together with the settings subject; API approved by the user 2026-10-03 (proposal revision 1 unchanged; see the change index and api-step.md); implementation authorized)\
Functional specification: [functional.md](functional.md), FR-APP-MENU-001 – 009 (FR-APP-MENU-007 added by issue #215; see [Issue #215](#issue-215-escape-right-after-opening-fr-app-menu-007); FR-APP-MENU-008 added by issue #223; see [Issue #223](#issue-223-subject-index-status-fr-app-menu-008); FR-APP-MENU-009 added by issue #225; see [Issue #225](#issue-225-arrow-keys-right-after-opening-fr-app-menu-009))\
Companion: [LeBoncoin session settings](../leboncoin-session-settings/technical.md) (the modal this menu opens)

## Requirement mapping

`MENU-*` abbreviates `FR-APP-MENU-*`.

| Technical ID | Functional IDs | Technical solution | Files expected to change | Verification |
| --- | --- | --- | --- | --- |
| TS-APP-MENU-001 | MENU-001, MENU-005 | `AppMenu` component in the existing `HeaderGlobalBar`, after the **Add item** button, on every page | `src/components/AppMenu.tsx` (new), `src/App.tsx`, `src/index.css` (only if alignment needs a class) | Playwright: icon visible on list, detail, loading and error states; 400 px layout |
| TS-APP-MENU-002 | MENU-002, MENU-003, MENU-004 | Carbon `OverflowMenu` with `UserAvatar` icon and one `OverflowMenuItem` “Settings”; selecting it opens `SettingsModal` | same | Playwright: click and Enter/Space open; Escape/outside click close; one item; selection opens modal |
| TS-APP-MENU-003 | MENU-004 | Focus return to the trigger via a ref passed to the modal's `launcherButtonRef` | `src/App.tsx`, `src/components/SettingsModal.tsx` | Playwright: focus on trigger after Escape and after modal close |
| TS-APP-MENU-004 | MENU-006 | Carbon modals are focus-trapped and cover the header; no extra code | — | Playwright: with Add item modal open, the trigger is not focusable/clickable |
| TS-APP-MENU-005 | MENU-007, MENU-004 | `AppMenu` controls the `OverflowMenu` `open` prop with its own state; an `onKeyDown` handler on the trigger closes the menu on Escape and moves focus on ArrowDown/ArrowUp when these keys are pressed before focus has moved into the menu | `src/components/AppMenu.tsx` (Escape already on master, PR #214; arrow keys added by issue #215) | Playwright TC-001, TC-002, TC-003 (issue #215) |
| TS-APP-MENU-007 | MENU-009, MENU-004, MENU-003 | No code change: the arrow-key branch of `handleTriggerKeyDown` (TS-APP-MENU-005) already gives the behaviour; throttled e2e tests now cover it | `tests/leboncoin-session-settings.spec.ts`, `tests/app-information.spec.ts` | Playwright TC-001 – TC-005 (issue #225) |

## Frontend

`src/components/AppMenu.tsx`:

```tsx
interface Props {
  onOpenSettings: () => void;
  triggerRef: React.RefObject<HTMLButtonElement | null>;
}
```

Renders Carbon `OverflowMenu` (installed `@carbon/react` 1.117.0 exports it) with:

- `renderIcon={UserAvatar}` from `@carbon/icons-react` (profile-style appearance, FR-APP-MENU-001).
- `aria-label="Application menu"` and `iconDescription="Application menu"` — describes the menu, never a signed-in user (FR-APP-MENU-004). Carbon sets `aria-haspopup`/`aria-expanded` on the trigger, gives it a focus ring, opens on click/Enter/Space, moves focus into the menu, closes on Escape/outside click and returns focus to the trigger.
- `flipped` so the menu aligns to the right edge and stays on screen at about 400 px (FR-APP-MENU-005); `size="lg"` to match the 48 px header height.
- `ref={triggerRef}` (the component forwards the ref to its trigger button in this Carbon version; the developer confirms with the installed types or Carbon MCP before relying on it, otherwise wraps a `HeaderGlobalAction` + `OverflowMenu` alternative only after reporting back).
- One child: `<OverflowMenuItem itemText="Settings" onClick={onOpenSettings} />` (FR-APP-MENU-003). Carbon closes the menu on selection.

Header styling: the header uses Carbon's default dark UI-shell styles. If the icon colour/alignment does not match the header, add one class in `src/index.css` using Carbon tokens only (e.g. `fill: var(--cds-icon-on-color)`); no hard-coded colours or spacing.

`src/App.tsx`: `const [settingsOpen, setSettingsOpen] = useState(false); const menuButtonRef = useRef<HTMLButtonElement>(null);` Inside `HeaderGlobalBar`, after the existing **Add item** `Button`: `<AppMenu onOpenSettings={() => setSettingsOpen(true)} triggerRef={menuButtonRef} />`. The settings modal receives `launcherButtonRef={menuButtonRef}`. The header is outside the page branches, so the control appears on the list, detail, loading, error and missing-item views (FR-APP-MENU-001). No route, URL, server request or browser storage is involved; opening/closing the menu has no side effects.

FR-APP-MENU-006: Carbon `Modal` renders an overlay above the header with a focus trap, so the trigger is unreachable while the Add item, Delete or Settings modal is open. No additional state is needed.

## Backend

Not affected: the menu needs no data and calls no endpoint.

## API

Not affected: no request is made by the menu. The settings modal it opens uses the pending settings endpoints (see the settings subject).

## Scope and refactoring

Files: new `src/components/AppMenu.tsx`; edits in `src/App.tsx`; optionally one class in `src/index.css`. No new dependency (icons and components are installed). No refactoring proposed; extracting the whole header into its own component is not needed for this change.

## Verification and unresolved questions

- `npm run build` (TypeScript + Vite).
- Playwright spec (shared with the settings subject, `tests/leboncoin-session-settings.spec.ts`, mocked API): trigger visible on list and detail pages and while the list request is pending or failed; accessible name “Application menu” and no user wording; `aria-expanded` toggles; keyboard open with Enter and Space; exactly one menu item “Settings”; Escape closes and focuses the trigger; outside click closes; selecting Settings opens the dialog named “Settings”; closing the dialog returns focus to the trigger; with the Add item dialog open, clicking at the trigger position does not open the menu; at 400×800 the trigger is visible, does not overlap the brand or Add item button (bounding boxes), the menu fits in the viewport, and the page has no horizontal scroll.
- Manual QA: screen reader announcement of the trigger and menu (record as a limitation if unavailable).

No unresolved question.

## Issue #215: Escape right after opening (FR-APP-MENU-007)

### Cause

In the installed `@carbon/react` (1.117), `OverflowMenu` handles Escape and Tab (`handleKeyPress`) only on the menu body (`ul role="menu"`), and arrow keys only on the menu body and its entries. Focus moves into the menu only after `FloatingMenu` has placed it, and only if focus is not already inside the menu. An Escape pressed in between targets the trigger button, which had no Escape handling, so the menu stayed open. An ArrowDown pressed in between is lost the same way: focus then lands on the first entry, and a following Enter selects **Information** instead of the second entry (review R1-01). Under load (full e2e suite) this showed up as intermittent failures of the existing "header menu: …" test.

### Frontend (TS-APP-MENU-005)

The Escape fix is already on master (PR #214, commit `542bedc`, merged 2026-11-23) in `src/components/AppMenu.tsx`; issue #215 adds the arrow keys to the same handler:

- `const [open, setOpen] = useState(false)` passed as `open={open}`, with `onOpen={() => setOpen(true)}` and `onClose={() => setOpen(false)}`. Carbon keeps its internal state, syncs it from the prop in an effect, and reports its own changes (click, outside click, item selection, Escape inside the menu) through `onOpen`/`onClose`, so both stay equal.
- `onKeyDown={handleTriggerKeyDown}`: Carbon spreads unknown props onto the trigger `IconButton` (before its own `onClick`, so the handler is not overridden). On Escape it sets `open` to `false`. Focus is already on the trigger, so it stays there; `aria-expanded` follows Carbon's state and becomes `false`.
- ArrowDown/ArrowUp on the trigger while the menu is rendered (found through the trigger's `aria-controls`): focus moves to the entry the key would reach from the first entry, where Carbon puts focus, so the result matches the same keys pressed slowly. That is the second entry for ArrowDown and the last entry for ArrowUp (wrapping, disabled entries skipped). `FloatingMenu` then finds focus inside the menu and leaves it there. While the menu is closed these keys keep their default behaviour.
- Escape once focus is inside the menu: unchanged Carbon path (`closeMenuOnEscape` closes and focuses the trigger, then `onClose` updates the state). Escape while the menu is closed is a no-op.
- Quick succession: Enter/Space on the trigger is a discrete event. React 19 renders it synchronously and flushes its passive effects (Carbon's `onOpen` call, then the prop sync) before the next key event, so each key acts on the state the previous one left. A focus move that `FloatingMenu` scheduled for a menu closed in the meantime targets an unmounted element and has no effect.
- No route, URL, server request or browser storage; accessibility attributes, focus ring and entries unchanged.

Developer work: the arrow keys on the trigger (review R1-01). Any fix stays inside `src/components/AppMenu.tsx`: no change to Carbon, no new dependency. Residual risk: the fix depends on Carbon forwarding `onKeyDown` to the trigger; TC-001 guards it on Carbon upgrades.

### Backend and API

Not affected: the menu calls no endpoint. `API_SPECIFICATION.md` is unchanged.

### Tests

New Playwright specs would need an entry in `testMatch` in `playwright.config.ts`, which this pipeline cannot change. The cases therefore go into specs that already run:

| Test case | Test file |
| --- | --- |
| TC-001 | `tests/leboncoin-session-settings.spec.ts` (open mode; the existing `gets` counter and unexpected-request check prove that no request is made); the protected-mode corner case (regular user) in `tests/app-information.spec.ts`, whose `mockApi` serves user and admin sessions |
| TC-002 | `tests/leboncoin-session-settings.spec.ts` |
| TC-003 | `tests/leboncoin-session-settings.spec.ts`: the existing "header menu: …" test plus a new test for the cases it lacks (Escape after the first entry is focused, click then Escape, Escape while closed) |

Test names start with the TC ID and `(#215)`, for example `TC-001 (#215) Escape at once after Enter closes the menu`, so they are not confused with other issues' `TC-001`. Slow-machine cases use `const cdp = await context.newCDPSession(page); await cdp.send("Emulation.setCPUThrottlingRate", { rate: 6 })`, which works in Chromium (the only project). Key sequences with "no wait" are consecutive `page.keyboard.press` calls; assertions come after the last key.

### Pipeline note

The source fix reached master before this change. The new tests are therefore expected to pass on the base branch, and the TDD red gate ("the tests pass before implementing") may stop the run. The owner decides how to proceed, for example by merging the regression tests by hand. Tests must not be weakened or made to fail on purpose to satisfy the gate.

## Issue #223: subject index status (FR-APP-MENU-008)

Documentation maintenance, planned together with the application-information subject: the design, files, tests and stage duties are in [Issue #223](../app-information/technical.md#issue-223-subject-index-and-status-line-fr-app-info-015-to-017) of the application-information technical specification. No behaviour change.

| Technical ID | Functional IDs | Technical solution | Files | Verification |
| --- | --- | --- | --- | --- |
| TS-APP-MENU-006 | FR-APP-MENU-008, FR-APP-INFO-017 | The Status cell of the **Header menu** row in section 15 of `doc/FUNCTIONAL_SPECIFICATIONS.md` is `implemented`; the row's link, the row order and every other line stay as on `master`. | `doc/FUNCTIONAL_SPECIFICATIONS.md` (one row, already edited by the functional specification stage) | TC-001, TC-003, TC-005, TC-006 |

Frontend, backend and API: not affected. `src/components/AppMenu.tsx`, its tests and `API_SPECIFICATION.md` are unchanged.

Implemented: TS-APP-MENU-006 (checks and gates pass, see the application-information section).

## Issue #225: arrow keys right after opening (FR-APP-MENU-009)

Review R2-01 on PR #221: the arrow-key branch of `handleTriggerKeyDown` (`focusEntryFromTrigger`, TS-APP-MENU-005) is not exercised by any test. The existing Enter, ArrowDown, Enter sequences run without throttling and pass even without the fix when Carbon has already focused the first entry. No test presses ArrowUp.

| Technical ID | Functional IDs | Technical solution | Files | Verification |
| --- | --- | --- | --- | --- |
| TS-APP-MENU-007 | FR-APP-MENU-009, FR-APP-MENU-004, FR-APP-MENU-003 | Throttled e2e tests (CPU rate 6, 5 rounds, keys pressed with no wait) for Enter, ArrowDown, Enter and for Enter, ArrowUp, in open mode and for a regular user, plus an unthrottled reference test with a pause before the arrow key. No production code change. | `tests/leboncoin-session-settings.spec.ts`, `tests/app-information.spec.ts` | TC-001 – TC-005 (issue #225) |

### Frontend

Not affected: `src/components/AppMenu.tsx` already does what FR-APP-MENU-009 requires (ArrowDown on the trigger focuses the second entry, ArrowUp the last, with wrap-around and disabled entries skipped). No component, state, accessibility attribute or style changes. If a new test fails on the base branch, the developer fixes `AppMenu.tsx` within TS-APP-MENU-005 (no Carbon change, no new dependency) and records the cause here.

### Backend and API

Not affected: the menu calls no endpoint. `API_SPECIFICATION.md` is unchanged. The Settings modal's own loads (`GET /api/v1/settings/leboncoin-session`, `GET /api/v1/settings/claude-token`, FR-LBC-SET-002) are mocked as they already are.

### Tests

Names start with the TC ID and `(#225)`, for example `TC-001 (#225) slow machine: Enter, ArrowDown, Enter at once opens Settings in each of 5 rounds`. "No wait" means consecutive `page.keyboard.press` calls; assertions come after the last key of a round. Between rounds, assertions and waits are allowed (they bring the page back to "menu closed, focus on the control"). Throttled tests call `test.setTimeout(90_000)`, as the slow TC-001 (#215) does.

| Test case | Test file | Design |
| --- | --- | --- |
| TC-001 | `tests/leboncoin-session-settings.spec.ts` | `mockApi(context)`, `listReady`, `throttleCpu`, focus `menuTrigger`. Each round: Enter, ArrowDown, Enter; the dialog "Settings" is visible and no `menuitem` is shown; wait until the dialog has loaded (`sessionField(page)` enabled), press Escape; the dialog is hidden and `expectMenuClosed`. After 5 rounds: URL `/`, `api.gets` is 5, and the recorded requests, sorted, are 5 × `GET /api/v1/settings/claude-token` and 5 × `GET /api/v1/settings/leboncoin-session` (the `afterEach` unexpected-request check also applies). |
| TC-002 | `tests/app-information.spec.ts` | `mockApi(context, [userSession])`, page loaded and `networkidle`, then record `${method} ${pathname}` of every `/api/` request, `throttleCpu`, focus the control. Each round as in TC-001: the dialog "Settings" is visible, it is the only visible `[role=dialog]` (not Information), no `menuitem`; once its "LeBonCoin session" textbox is enabled, Escape; dialog hidden, control focused, `aria-expanded="false"`. After 5 rounds: no `POST /api/v1/auth/logout` and only the Settings loads were requested; Enter reopens the menu with `["Information", "Settings", "Log out"]`, Escape closes it; URL `/`. |
| TC-003 | `tests/leboncoin-session-settings.spec.ts` | Setup as TC-001. Each round: Enter, ArrowUp; `expectMenuOpen` and the `menuitem` "Settings" is focused; Escape; `expectMenuClosed`; no visible `[role=dialog]`. After 5 rounds: URL `/`, recorded requests `[]`, `api.gets` 0. |
| TC-004 | `tests/app-information.spec.ts` | Setup as TC-002. Each round: Enter, ArrowUp; the entries are `["Information", "Settings", "Log out"]`, `aria-expanded="true"` and "Log out" is focused; Escape; no `menuitem`, control focused, `aria-expanded="false"`. After 5 rounds: recorded requests `[]` (no logout), URL `/`. |
| TC-005 | `tests/leboncoin-session-settings.spec.ts` | No throttling. Enter; wait until "Information" is focused; ArrowDown; "Settings" is focused; Enter opens the dialog "Settings"; once loaded, Escape and `expectMenuClosed`. Then Enter; wait until "Information" is focused; ArrowUp; "Settings" is focused; Escape; `expectMenuClosed`. With two entries, one step lands on Settings and a second step would wrap back to Information, so the focus check proves the key was handled once. `api.gets` 1, URL `/`. |

Test helper changes:

- `tests/app-information.spec.ts`: its `mockApi` answers 500 to unknown paths. Add two routes so the Settings modal loads normally: `GET /api/v1/settings/leboncoin-session` → `{ session: { saved: false, hint: null, revision: 0, updatedAt: null, status: "none", expiresAt: null, revokedAt: null, lastAttempt: null } }` and `GET /api/v1/settings/claude-token` → `{ claudeToken: { saved: false, hint: null, updatedAt: null, lastRejectedAt: null } }`. No existing test in the file opens Settings, so none is affected. A local helper for the shared setup of TC-002 and TC-004 is allowed; reuse the file's `throttleCpu`.
- `tests/leboncoin-session-settings.spec.ts`: reuse `throttleCpu`, `listReady`, `expectMenuOpen`, `expectMenuClosed`, `menuTrigger` and `sessionField`; place the new tests after the issue #215 tests. No helper change.

Waiting for the loaded modal before Escape keeps the close step independent of the loading state; it is not part of the key sequence under test.

Fails without the fix (not committed): reduce `handleTriggerKeyDown` in `src/components/AppMenu.tsx` to its Escape line, run `npx playwright test -g "\(#225\)"`, and check that TC-001 – TC-004 fail (TC-005 passes: it presses the arrow key once focus is in the menu). Restore the file (`git diff -- src` empty). The test author does this to validate the tests; the developer repeats it and reports the result in the pull request. If a throttled test still passes with the reduced handler, report it; do not change the rate or the round count set by the test cases.

The existing header-menu tests (TC-001 – TC-003 (#215), "header menu: …", the Enter, ArrowDown, Enter steps in other tests) stay unchanged and must pass.

### Files to change

- Tests stage: `tests/leboncoin-session-settings.spec.ts` (TC-001, TC-003, TC-005), `tests/app-information.spec.ts` (two mock routes, TC-002, TC-004).
- Development stage: no source change expected. Mark FR-APP-MENU-009 implemented in the Status line of `functional.md` and TS-APP-MENU-007 implemented in this file once the gates pass. `doc/FUNCTIONAL_SPECIFICATIONS.md` is already updated (FR-23, by the functional specification stage); its section 15 **Header menu** row stays `implemented`.
- Not changed: `src/`, `API_SPECIFICATION.md`, `playwright.config.ts` (both specs are already in `testMatch`), dependencies.

### Refactoring

None.

### Manual follow-up (pipeline)

The behaviour shipped with issue #215, so the new tests are expected to pass on the base branch, and the TDD red gate (`gate_red`: "the tests pass before implementing") will stop the run. This is the situation described in the [issue #215 pipeline note](#pipeline-note). The owner decides how to proceed, for example by accepting the tests past the red gate by hand. Tests must not be weakened, and must not be made to fail on purpose, to satisfy the gate. No test depends on this follow-up.

Open: TS-APP-MENU-007 stays specified; the regression check is not met. With `handleTriggerKeyDown` reduced to its Escape line, TC-001 – TC-005 (#225) all still pass, as the tests stage also found (change folder `tests-note.md`). Playwright's `page.keyboard.press` returns only after the renderer has handled the key, so Carbon has already focused **Information** when the arrow key arrives, even at CPU rate 6, and the trigger's arrow-key branch never runs. The tests check FR-APP-MENU-009 but do not catch a regression in that branch. Returned to the tests stage (change folder `test-problem.md`).

## Revision note

2026-10-03, review amendment REV-SET-001: status updated to record the API approval of 2026-10-03.

2026-11-23, issue #215: TS-APP-MENU-005 added for FR-APP-MENU-007 (Escape right after opening). The Escape code was already on master (PR #214); review R1-01 added ArrowDown/ArrowUp on the trigger.

2026-11-28, issue #223: TS-APP-MENU-006 added for FR-APP-MENU-008 (subject index status, documentation only).

2026-11-28, issue #225: TS-APP-MENU-007 added for FR-APP-MENU-009 (throttled e2e tests for ArrowDown/ArrowUp right after opening; no production code change).
